package chaos_tcp

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

type Connection struct {
	name           string
	server         *Server
	address        *Address
	conn           *net.TCPConn
	receiveChannel chan *Message
	sendChannel    chan *Message
	running        bool
	wg             sync.WaitGroup
	lock           sync.Mutex
}

func NewConnection(server *Server, conn *net.TCPConn, address *Address) *Connection {
	con := new(Connection)
	con.server = server
	con.conn = conn
	con.address = address
	con.sendChannel = make(chan *Message, 10)
	con.receiveChannel = make(chan *Message, 10)
	con.running = true
	return con
}

func (c *Connection) startWorker() {
	go c.recv()
	go c.handle()
	go c.send()
}

func (c *Connection) Close() {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.running {
		c.running = false
		c.conn.Close()
		c.wg.Wait()
		close(c.sendChannel)
		close(c.receiveChannel)
	}
}

func (c *Connection) recv() {
	c.wg.Add(1)
	defer c.wg.Done()

	for c.running {
		time.Sleep(time.Second)
		msg := Message{}

		//c.conn.SetReadDeadline(time.Now().Add(readTimeout))

		err := read(c.conn, &msg)
		if err != nil {
			switch err {
			case io.EOF /*errRecvEOF, errRemoteForceDisconnect*/ :
				fmt.Errorf("this client connect is close: %s.", err.Error())
				c.server.closeConnection(c.address)
				c.Close()
				return
			default:
				fmt.Errorf("recv msg err: %s", err.Error())
			}

			continue
		}

		// skip heart beat package
		//if msg.Type == heartbeat {
		//	continue
		//}

		c.receiveChannel <- &msg
	}
}

func (c *Connection) handle() {
	c.wg.Add(1)
	defer c.wg.Done()

	for c.running {
		select {
		case msg := <-c.receiveChannel:
			go func() {
				c.server.callOnReceive(c.address, msg)
			}()
		}
	}
}

func (c *Connection) send() {
	c.wg.Add(1)
	defer c.wg.Done()

	for c.running {
		select {
		case msg := <-c.sendChannel:
			data, err := pack(msg)
			if err != nil {
				fmt.Errorf("pack data address(%s) error:%s", c.address.GetAddress(), err.Error())
				continue
			}

			//c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			_, err = c.conn.Write(data.Bytes())
			if err != nil {
				// broken pipe, use of closed network connection, or other write error
				fmt.Errorf("send data to client address(%s) error:%s", c.address.GetAddress(), err.Error())
				c.server.closeConnection(c.address)
				c.Close()
				return
			}
		}
	}
}
