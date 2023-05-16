package rpc

import (
	"encoding/binary"
	"errors"
	"fmt"
	"go/token"
	"log"
	"math"
	"net"
	"reflect"
	"strings"
	"sync"

	"github.com/cloudwego/shmipc-go"
)

type ShmReader interface {
	ReadFromShm(shmipc.BufferReader) error
}

type ShmWriter interface {
	WriteToShm(shmipc.BufferWriter) error
}

type ShmReadWriter interface {
	ShmReader
	ShmWriter
}

// Precompute the reflect type for error and ShmReadWriter. Can't use error
// directly because Typeof takes an empty interface value. This is annoying.
var (
	typeOfError         = reflect.TypeOf((*error)(nil)).Elem()
	typeOfShmReadWriter = reflect.TypeOf((*ShmReadWriter)(nil)).Elem()
)

type methodType struct {
	method    reflect.Method
	ArgType   reflect.Type
	ReplyType reflect.Type
}

type service struct {
	name   string                 // name of service
	rcvr   reflect.Value          // receiver of methods for the service
	typ    reflect.Type           // type of the receiver
	method map[string]*methodType // registered methods
}

func (s *service) call(server *Server, stream *shmipc.Stream, mtype *methodType, argv, replyv reflect.Value) {
	function := mtype.method.Func
	// Invoke the method, providing a new value for the reply.
	returnValues := function.Call([]reflect.Value{s.rcvr, argv, replyv})
	// The return value for the method is an error.
	errmsg := ""
	if erri := returnValues[0].Interface(); erri != nil {
		errmsg = erri.(error).Error()
	}

	if err := server.sendResponse(stream, replyv, errmsg); err != nil {
		log.Print("rpc: error sending response: ", err.Error())
	}
}

// Server represents an RPC Server.
type Server struct {
	serviceMap sync.Map // map[string]*service
	conf       *shmipc.Config
}

// NewServer returns a new Server.
func NewServer(conf *shmipc.Config) *Server {
	return &Server{conf: conf}
}

// DefaultServer is the default instance of *Server.
var DefaultServer = NewServer(shmipc.DefaultConfig())

// Is this type exported?
func isExported(t reflect.Type) bool {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	// PkgPath will be non-empty even for an exported type,
	// so we need to check the type name as well.
	return token.IsExported(t.Name())
}

// Register publishes in the server the set of methods of the
// receiver value that satisfy the following conditions:
//   - exported method of exported type
//   - two arguments, both of exported type
//   - the second argument is a pointer
//   - one return value, of type error
//
// It returns an error if the receiver is not an exported type or has
// no suitable methods. It also logs the error using package log.
// The client accesses each method using a string of the form "Type.Method",
// where Type is the receiver's concrete type.
func (server *Server) Register(rcvr interface{}) error {
	return server.register(rcvr, "", false)
}

// RegisterName is like Register but uses the provided name for the type
// instead of the receiver's concrete type.
func (server *Server) RegisterName(name string, rcvr interface{}) error {
	return server.register(rcvr, name, true)
}

// logRegisterError specifies whether to log problems during method registration.
// To debug registration, recompile the package with this set to true.
const logRegisterError = true

func (server *Server) register(rcvr interface{}, name string, useName bool) error {
	s := new(service)
	s.typ = reflect.TypeOf(rcvr)
	s.rcvr = reflect.ValueOf(rcvr)
	sname := name
	if !useName {
		sname = reflect.Indirect(s.rcvr).Type().Name()
	}
	if sname == "" {
		s := "rpc.Register: no service name for type " + s.typ.String()
		log.Print(s)
		return errors.New(s)
	}
	if !useName && !token.IsExported(sname) {
		s := "rpc.Register: type " + sname + " is not exported"
		log.Print(s)
		return errors.New(s)
	}
	s.name = sname

	// Install the methods
	s.method = suitableMethods(s.typ, logRegisterError)

	if len(s.method) == 0 {
		str := ""

		// To help the user, see if a pointer receiver would work.
		method := suitableMethods(reflect.PointerTo(s.typ), false)
		if len(method) != 0 {
			str = "rpc.Register: type " + sname + " has no exported methods of suitable type (hint: pass a pointer to value of that type)"
		} else {
			str = "rpc.Register: type " + sname + " has no exported methods of suitable type"
		}
		log.Print(str)
		return errors.New(str)
	}

	if _, dup := server.serviceMap.LoadOrStore(sname, s); dup {
		return errors.New("rpc: service already defined: " + sname)
	}
	return nil
}

// suitableMethods returns suitable Rpc methods of typ. It will log
// errors if logErr is true.
func suitableMethods(typ reflect.Type, logErr bool) map[string]*methodType {
	methods := make(map[string]*methodType)
	for m := 0; m < typ.NumMethod(); m++ {
		method := typ.Method(m)
		mtype := method.Type
		mname := method.Name
		// Method must be exported.
		if !method.IsExported() {
			continue
		}
		// Method needs three ins: receiver, *args, *reply.
		if mtype.NumIn() != 3 {
			if logErr {
				log.Printf("rpc.Register: method %q has %d input parameters; needs exactly three\n", mname, mtype.NumIn())
			}
			continue
		}
		// First arg need not be a pointer.
		argType := mtype.In(1)
		if !isExported(argType) {
			if logErr {
				log.Printf("rpc.Register: argument type of method %q is not exported: %q\n", mname, argType)
			}
			continue
		}
		// Arg type must implement ShmReadWriter.
		if !argType.Implements(typeOfShmReadWriter) {
			if logErr {
				log.Printf("rpc.Register: argument type of method %q does not implement ShmReadWriter: %q\n", mname, argType)
			}
			continue
		}
		// Second arg must be a pointer.
		replyType := mtype.In(2)
		if replyType.Kind() != reflect.Pointer {
			if logErr {
				log.Printf("rpc.Register: reply type of method %q is not a pointer: %q\n", mname, replyType)
			}
			continue
		}
		// Reply type must be exported.
		if !isExported(replyType) {
			if logErr {
				log.Printf("rpc.Register: reply type of method %q is not exported: %q\n", mname, replyType)
			}
			continue
		}
		// Reply type must implement ShmReadWriter.
		if !replyType.Implements(typeOfShmReadWriter) {
			if logErr {
				log.Printf("rpc.Register: reply type of method %q does not implement ShmReadWriter: %q\n", mname, replyType)
			}
			continue
		}
		// Method needs one out.
		if mtype.NumOut() != 1 {
			if logErr {
				log.Printf("rpc.Register: method %q has %d output parameters; needs exactly one\n", mname, mtype.NumOut())
			}
			continue
		}
		// The return type of the method must be error.
		if returnType := mtype.Out(0); returnType != typeOfError {
			if logErr {
				log.Printf("rpc.Register: return type of method %q is %q, must be error\n", mname, returnType)
			}
			continue
		}
		methods[mname] = &methodType{method: method, ArgType: argType, ReplyType: replyType}
	}
	return methods
}

func (server *Server) Accept(lis net.Listener) {
	for {
		conn, err := lis.Accept()
		if err != nil {
			log.Print("rpc.Serve: accept:", err.Error())
			return
		}
		go server.ServeConn(conn)
	}
}

func (server *Server) ServeConn(conn net.Conn) {
	defer conn.Close()

	shmServer, err := shmipc.Server(conn, server.conf)
	if err != nil {
		log.Print("rpc.ServeConn: error creating shmipc server: ", err.Error())
		return
	}
	defer shmServer.Close()

	for {
		stream, err := shmServer.AcceptStream()
		if err != nil {
			log.Print("rpc.Serve: error accepting shmipc stream:", err.Error())
			break
		}
		go server.ServeStream(stream)
	}
}

func (server *Server) ServeStream(stream *shmipc.Stream) {
	service, mtype, argv, replyv, err := server.readRequest(stream.BufferReader())
	if err != nil {
		if err := server.sendResponse(stream, replyv, err.Error()); err != nil {
			log.Print("rpc.ServeRequest: error sending response: ", err.Error())
		}
		return
	}

	service.call(server, stream, mtype, argv, replyv)
}

func (server *Server) readRequest(r shmipc.BufferReader) (svc *service, mtype *methodType, argv, replyv reflect.Value, err error) {
	data, err := r.ReadBytes(2)
	if err != nil {
		err = fmt.Errorf("rpc: error read service method's len: %s", err)
		return
	}
	serviceMethodLen := binary.BigEndian.Uint16(data)

	data, err = r.ReadBytes(int(serviceMethodLen))
	if err != nil {
		err = fmt.Errorf("rpc: error read service method: %s", err)
		return
	}
	serviceMethod := string(data)

	dot := strings.LastIndex(serviceMethod, ".")
	if dot < 0 {
		err = errors.New("rpc: service/method request ill-formed: " + serviceMethod)
		return
	}
	serviceName := serviceMethod[:dot]
	methodName := serviceMethod[dot+1:]

	// Look up the request.
	svci, ok := server.serviceMap.Load(serviceName)
	if !ok {
		err = errors.New("rpc: can't find service " + serviceMethod)
		return
	}
	svc = svci.(*service)
	mtype = svc.method[methodName]
	if mtype == nil {
		err = errors.New("rpc: can't find method " + serviceMethod)
		return
	}

	// Decode the argument value.
	argIsValue := false // if true, need to indirect before calling.
	if mtype.ArgType.Kind() == reflect.Pointer {
		argv = reflect.New(mtype.ArgType.Elem())
	} else {
		argv = reflect.New(mtype.ArgType)
		argIsValue = true
	}

	returnValues := argv.MethodByName("ReadFromShm").Call([]reflect.Value{reflect.ValueOf(r)})
	if erri := returnValues[0].Interface(); erri != nil {
		err = fmt.Errorf("rpc: can't read from shared memory: %s ", erri)
		return
	}

	if argIsValue {
		argv = argv.Elem()
	}

	replyv = reflect.New(mtype.ReplyType.Elem())
	return
}

func (server *Server) sendResponse(stream *shmipc.Stream, replyv reflect.Value, errmsg string) error {
	w := stream.BufferWriter()

	errmsgBytes := []byte(errmsg)
	if len(errmsgBytes) > math.MaxUint32 {
		errmsgBytes = errmsgBytes[:math.MaxUint32]
	}
	errmsgLen := len(errmsgBytes)

	data, err := w.Reserve(4)
	if err != nil {
		return fmt.Errorf("rpc: can't write error message's len: %s ", err)
	}
	binary.BigEndian.PutUint32(data, uint32(errmsgLen))

	if errmsgLen > 0 {
		if _, err := w.WriteBytes(errmsgBytes); err != nil {
			return fmt.Errorf("rpc: can't write error message: %s ", err)
		}

		return nil
	}

	returnValues := replyv.MethodByName("WriteToShm").Call([]reflect.Value{reflect.ValueOf(w)})
	if erri := returnValues[0].Interface(); erri != nil {
		return fmt.Errorf("rpc: can't write to shared memory: %s ", erri)
	}

	if err := stream.Flush(false); err != nil {
		return fmt.Errorf("rpc: can't flush response to peer: %s", err)
	}

	return nil
}
