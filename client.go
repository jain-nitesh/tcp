package chaos_tcp

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

type ClientReceiveHandler func(receive *Message)

type Client struct {
	remoteAddress *Address
	localAddress  *Address
	connection    net.Conn
	onReceive     ClientReceiveHandler
	running       bool
	connected     bool
	wg            sync.WaitGroup
	connectedLock sync.Mutex
}

func (c *Client) connectServer(address string) error {
	c.connectedLock.Lock()
	defer c.connectedLock.Unlock()

	if c.connected {
		return nil
	}
	c.remoteAddress = NewAddr(address)

	tcpAdd, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		return fmt.Errorf("get address info error:%s", err.Error())
	}

	conn, err := net.DialTCP("tcp", nil, tcpAdd)
	if err != nil {
		return fmt.Errorf("connect server error:%s", err.Error())
	}

	if err = conn.SetKeepAlive(true); err != nil {
		fmt.Errorf("set keep alive err: %s", err.Error())
	}
	if err = conn.SetKeepAlivePeriod(5 * time.Second); err != nil {
		fmt.Errorf("set keep alive period err: %s", err.Error())
	}

	c.connection = conn
	c.connected = true
	c.localAddress = NewAddr(conn.LocalAddr().String())

	fmt.Sprintf("connect server success, address=%s.", address)
	go c.recv()
	//go c.heartbeat()

	return nil
}

func (c *Client) reconnect() {
	c.wg.Add(1)
	defer c.wg.Done()

	var currentConnectStatus error
	for c.running {
		time.Sleep(time.Second * 3)

		err := c.connectServer(c.remoteAddress.GetAddress())
		// filtering duplicate logs
		if err != nil {
			if currentConnectStatus == nil || err.Error() != currentConnectStatus.Error() {
				currentConnectStatus = err
				fmt.Errorf(err.Error())
			}
		}
	}
}

func (c *Client) recv() {
	c.wg.Add(1)
	defer c.wg.Done()

	for c.running && c.connected {
		msg := Message{}
		if err := read(c.connection, &msg); err != nil {
			switch err {
			case io.EOF /*errRecvEOF, errRemoteForceDisconnect*/ :
				fmt.Errorf("this client connect disconnect: %s", err.Error())
				c.connected = false
				break
			default:
				fmt.Errorf("recv msg err: %s", err.Error())
			}
			continue
		}

		c.callOnReceive(&msg)
	}
}

func (c *Client) callOnReceive(m *Message) {
	go func() {
		if c.onReceive != nil {
			c.onReceive(m)
		}
	}()
}

func NewClient(address string) *Client {
	client := new(Client)
	client.connectServer(address)
	client.running = true
	go client.reconnect()
	return client
}

func (c *Client) OnReceive(handle ClientReceiveHandler) {
	c.onReceive = handle
}

func (c *Client) Send(msg *Message) error {
	if c.connection != nil {
		// pack data
		buf, err := pack(msg)
		if err != nil {
			return fmt.Errorf("pack data error:%s", err.Error())
		}

		// send data
		//c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
		_, err = c.connection.Write(buf.Bytes())
		if err != nil {
			// broken pipe, use of closed network connection, or other write error.
			// if connect is close, then reconnect function will connect to server later.
			c.connected = false
			return fmt.Errorf("send data error:%s", err.Error())
		}
	}
	return nil
}

func (c *Client) Close() {
	c.running = false
	if c.connection != nil { // fix: connect fail, then c.conn is nil.
		c.connection.Close()
	}
	c.wg.Wait()
}
