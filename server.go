package tcp

import (
	"fmt"
	"net"
	"sync"
)

func init() {
	//	TODO setup
}

type ServerReceiveHandler func(address *Address, msg *Message)

type ServerConnectHandler func(conn *net.TCPConn, address *Address)

type ServerDisconnectHandler func(addr *Address)

type Server struct {
	listener     *net.TCPListener
	addToConnMap map[string]*Connection
	onReceive    ServerReceiveHandler
	onConnect    ServerConnectHandler
	onDisconnect ServerDisconnectHandler
	lock         sync.RWMutex
	running      bool
}

func NewServer() *Server {
	s := new(Server)
	s.addToConnMap = make(map[string]*Connection)
	s.running = true
	return s
}

func (s *Server) OnReceive(handle ServerReceiveHandler) {
	s.onReceive = handle
}

func (s *Server) callOnReceive(addr *Address, request *Message) {
	go func() {
		if s.onReceive != nil {
			s.onReceive(addr, request)
		}
	}()
}

func (s *Server) Send(addr string, message *Message) error {
	if connection, ok := s.addToConnMap[addr]; !ok {
		return fmt.Errorf("connection does not exist for address [%s]", addr)
	} else {
		go func() {
			connection.sendChannel <- message
		}()
	}
	return nil
}

func (s *Server) OnConnect(handle ServerConnectHandler) {
	s.onConnect = handle
}

func (s *Server) callOnConnect(conn *net.TCPConn, addr *Address) {
	go func() {
		if s.onConnect != nil {
			s.onConnect(conn, addr)
		}
	}()
}

func (s *Server) OnDisconnect(handle ServerDisconnectHandler) {
	s.onDisconnect = handle
}

func (s *Server) callOnDisconnect(addr *Address) {
	go func() {
		if s.onDisconnect != nil {
			s.onDisconnect(addr)
		}
	}()
}

func (s *Server) Close() {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.running = false
	if s.listener != nil {
		s.listener.Close()
	}

	var wg sync.WaitGroup
	for addr, connection := range s.addToConnMap {
		go func() {
			wg.Add(1)
			connection.Close()
			delete(s.addToConnMap, addr)
			wg.Done()
		}()
	}

	wg.Wait()
}
func (s *Server) closeConnection(addr *Address) {
	s.callOnDisconnect(addr)

	if connection, ok := s.addToConnMap[addr.GetAddress()]; !ok {
		connection.Close()
		delete(s.addToConnMap, addr.GetAddress())
	}

}
func (s *Server) Run(address string) {

	tcpAddr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		fmt.Errorf("the server listen address err:%s", err.Error())
		return
	}

	tcpListener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		fmt.Errorf("tcp server listen addr(%s) error:%s", address, err.Error())
		return
	}

	s.listener = tcpListener

	for s.running {
		tcpConn, err := s.listener.AcceptTCP()
		if err != nil {
			fmt.Errorf("tcp server accept one client error:%s", err.Error())
			continue
		}

		addr := NewAddr(tcpConn.RemoteAddr().String())

		connection := NewConnection(s, tcpConn, addr)

		if s.addToConnMap == nil {
			s.addToConnMap = make(map[string]*Connection)
		}

		s.addToConnMap[addr.GetAddress()] = connection

		s.callOnConnect(tcpConn, addr)

		connection.startWorker()

	}

}
