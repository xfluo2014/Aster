package networks

import (
	"blockEmulator/params"
	"bytes"
	"context"
	"io"
	"log"
	"math/rand"
	"net"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

var connMaplock sync.Mutex
var connectionPool = make(map[string]net.Conn, 0)

//var limiter *rate.Limiter
//var NodeBandwidth = 0

var NodeID = 0

var randomDelayGenerator *rand.Rand
var rateLimiterDownload *rate.Limiter
var rateLimiterUpload *rate.Limiter

// Define the latency, jitter and bandwidth here.
// Init tools.
func InitNetworkTools() {
	// avoid wrong params.
	if params.Delay < 0 {
		params.Delay = 0
	}
	if params.JitterRange < 0 {
		params.JitterRange = 0
	}
	if params.Bandwidth < 0 {
		params.Bandwidth = 0x7fffffff
	}

	// generate the random seed.
	randomDelayGenerator = rand.New(rand.NewSource(time.Now().UnixMicro()))
	// Limit the download rate
	rateLimiterDownload = rate.NewLimiter(rate.Limit(params.Bandwidth), params.Bandwidth)
	// Limit the upload rate
	rateLimiterUpload = rate.NewLimiter(rate.Limit(params.Bandwidth), params.Bandwidth)
}

//	func DynamicBandwidth() {
//		if !params.StartDynamicBandwidth {
//			return
//		}
//		//// Create a ticker that fires every 50 seconds
//		go func() {
//			nums := []int{100, 1, 100, 1, 100, 1, 100, 1}
//			index := 0
//			t := 50
//			ticker := time.NewTicker(time.Duration(t) * time.Second)
//			defer ticker.Stop()
//			first := 0
//			WriteBandwidthCSV(first, NodeBandwidth)
//
//			for {
//				<-ticker.C
//				first++
//				// Update the limiter's rate and burst size every 10 seconds
//				index++
//				NodeBandwidth = nums[index%len(nums)]
//				limiter.SetLimit(rate.Limit(NodeBandwidth * 1024 * 1024))
//				limiter.SetBurst(NodeBandwidth * 1024 * 1024)
//				fmt.Println("Updated limiter:", limiter.Limit(), limiter.Burst())
//				WriteBandwidthCSV(first, NodeBandwidth)
//			}
//		}()
//	}
func TcpDial(context []byte, addr string) {
	go func() {
		// simulate the delay
		thisDelay := params.Delay
		if params.JitterRange != 0 {
			thisDelay = randomDelayGenerator.Intn(params.JitterRange) - params.JitterRange/2 + params.Delay
		}
		time.Sleep(time.Millisecond * time.Duration(thisDelay))

		connMaplock.Lock()
		defer connMaplock.Unlock()

		var err error
		var conn net.Conn // Define conn here

		// if this connection is not built, build it.
		if c, ok := connectionPool[addr]; ok {
			if tcpConn, tcpOk := c.(*net.TCPConn); tcpOk {
				if err := tcpConn.SetKeepAlive(true); err != nil {
					delete(connectionPool, addr) // Remove if not alive
					conn, err = net.Dial("tcp", addr)
					if err != nil {
						log.Println("Reconnect error", err)
						return
					}
					connectionPool[addr] = conn
					go ReadFromConn(addr) // Start reading from new connection
				} else {
					conn = c // Use the existing connection
				}
			}
		} else {
			conn, err = net.Dial("tcp", addr)
			if err != nil {
				log.Println("Connect error", err)
				return
			}
			connectionPool[addr] = conn
			go ReadFromConn(addr) // Start reading from new connection
		}

		writeToConn(append(context, '\n'), conn, rateLimiterUpload)
	}()
}
func writeToConn(connMsg []byte, conn net.Conn, limiter *rate.Limiter) {
	// Wrap the connection with rateLimitedWriter
	rateLimitedConn := &rateLimitedWriter{writer: conn, limiter: limiter}

	// writing data to the connection
	_, err := rateLimitedConn.Write(connMsg)
	if err != nil {
		log.Println("Write error", err)
		return
	}
}

type rateLimitedWriter struct {
	writer  io.Writer
	limiter *rate.Limiter
}

// Write method waits for the rate limiter and then writes data
func (w *rateLimitedWriter) Write(p []byte) (int, error) {
	// Calculate the number of bytes to write and wait for the limiter to grant enough tokens
	n := len(p)
	for n > 0 {
		writeByteNum := w.limiter.Burst()
		if writeByteNum > n {
			writeByteNum = n
		}
		if err := w.limiter.WaitN(context.TODO(), writeByteNum); err != nil {
			return 0, err
		}
		n -= writeByteNum
	}
	// Actually write the data
	return w.writer.Write(p)
}

//func TcpDial(msg []byte, addr string) {
//	//time.Sleep(time.Duration(params.FollowerNum) * time.Millisecond * 3)
//	//time.Sleep(time.Millisecond * 15)
//	connMaplock.Lock()
//	defer connMaplock.Unlock()
//
//	var err error
//	var conn net.Conn // Define conn here
//	if c, ok := connectionPool[addr]; ok {
//		if tcpConn, tcpOk := c.(*net.TCPConn); tcpOk {
//			if err := tcpConn.SetKeepAlive(true); err != nil {
//				delete(connectionPool, addr) // Remove if not alive
//				conn, err = net.Dial("tcp", addr)
//				if err != nil {
//					log.Println("Reconnect error", err)
//					return
//				}
//				connectionPool[addr] = conn
//				go ReadFromConn(addr) // Start reading from new connection
//			} else {
//				conn = c // Use the existing connection
//			}
//		}
//	} else {
//		//if !params.IsFollower {
//		//	localAddr := &net.TCPAddr{
//		//		IP:   net.ParseIP("192.168.1.100"), // 替换为你的局域网 IP 地址
//		//		Port: 0,                            // 端口号设置为 0，让系统自动选择
//		//	}
//		//	atoi, _ := strconv.Atoi(strings.Split(addr, ":")[1])
//		//	remoteAddr := &net.TCPAddr{
//		//		IP:   net.ParseIP(strings.Split(addr, ":")[0]),
//		//		Port: atoi, // 目标服务器的端口号
//		//	}
//		//	net.DialIP("tcp", nil, localAddr, remoteAddr)
//		//}
//		//net.DialIP()
//		conn, err = net.Dial("tcp", addr)
//		if err != nil {
//			log.Println("Connect error", err)
//			return
//		}
//		connectionPool[addr] = conn
//		go ReadFromConn(addr) // Start reading from new connection
//	}
//
//	// Using bytes.Buffer for efficient data management
//	buffer := bytes.NewBuffer(msg)
//	buffer.WriteByte('\n')
//
//	//fmt.Println("message size", buffer.Len())
//
//	for buffer.Len() > 0 {
//		// Determine how much we can write at this moment
//		chunkSize := limiter.Burst()
//
//		//fmt.Println("read send message===================== chunkSize", chunkSize, "buffer.Len()", buffer.Len())
//
//		if chunkSize > buffer.Len() {
//			chunkSize = buffer.Len()
//		}
//
//		// Wait for permission to write chunkSize bytes
//		if err := limiter.WaitN(context.Background(), chunkSize); err != nil {
//			log.Println("Rate limiter error", err)
//			return
//		}
//
//		// Write the chunk
//		chunk := buffer.Next(chunkSize)
//		_, err = conn.Write(chunk)
//		if err != nil {
//			log.Println("Write error", err)
//			return
//		}
//	}
//
//	// _, err = conn.Write(append(context, '\n'))
//	// if err != nil {
//	// 	log.Println("Write error", err)
//	// 	return
//	// }
//}

// Broadcast sends a message to multiple receivers, excluding the sender.
func Broadcast(sender string, receivers []string, msg []byte) {
	for _, ip := range receivers {
		go TcpDial(msg, ip)
	}
}

// CloseAllConnInPool closes all connections in the connection pool.
func CloseAllConnInPool() {
	connMaplock.Lock()
	defer connMaplock.Unlock()

	for _, conn := range connectionPool {
		conn.Close()
	}
	connectionPool = make(map[string]net.Conn) // Reset the pool
}
func ReadFromConn(addr string) {
	conn := connectionPool[addr]

	// new a conn reader
	connReader := NewConnReader(conn, rateLimiterDownload)

	buffer := make([]byte, 1024)
	var messageBuffer bytes.Buffer

	for {
		n, err := connReader.Read(buffer)
		if err != nil {
			if err != io.EOF {
				log.Println("Read error for address", addr, ":", err)
			}
			break
		}

		// add message to buffer
		messageBuffer.Write(buffer[:n])

		// handle the full message
		for {
			message, err := readMessage(&messageBuffer)
			if err == io.ErrShortBuffer {
				// continue to load if buffer is short
				break
			} else if err == nil {
				// log the full message
				log.Println("Received from", addr, ":", message)
			} else {
				// handle other errs
				log.Println("Error processing message for address", addr, ":", err)
				break
			}
		}
	}
}

type rateLimitedReader struct {
	reader  io.Reader
	limiter *rate.Limiter
}

// Read method waits for the rate limiter and then reads data
func (r *rateLimitedReader) Read(p []byte) (int, error) {
	// Calculate the number of bytes to read and wait for the limiter to grant enough tokens
	n := len(p)
	for n > 0 {
		writeByteNum := r.limiter.Burst()
		if writeByteNum > n {
			writeByteNum = n
		}
		if err := r.limiter.WaitN(context.TODO(), n); err != nil {
			return 0, err
		}
		n -= writeByteNum
	}

	// Actually read the data
	return r.reader.Read(p)
}

func NewConnReader(conn net.Conn, rateLimiter *rate.Limiter) *rateLimitedReader {
	// Wrap the connection with rateLimitedReader
	return &rateLimitedReader{reader: conn, limiter: rateLimiter}
}

// ReadFromConn reads data from a connection.
//func ReadFromConn(addr string) {
//	conn := connectionPool[addr]
//
//	buffer := make([]byte, 1024)
//	var messageBuffer bytes.Buffer
//
//	for {
//		n, err := conn.Read(buffer)
//		if err != nil {
//			if err != io.EOF {
//				log.Println("Read error for address", addr, ":", err)
//			}
//			break
//		}
//
//		// add message to buffer
//		messageBuffer.Write(buffer[:n])
//
//		// handle the full message
//		for {
//			message, err := readMessage(&messageBuffer)
//			if err == io.ErrShortBuffer {
//				// continue to load if buffer is short
//				break
//			} else if err == nil {
//				// log the full message
//				log.Println("Received from", addr, ":", message)
//			} else {
//				// handle other errs
//				log.Println("Error processing message for address", addr, ":", err)
//				break
//			}
//		}
//	}
//}

func readMessage(buffer *bytes.Buffer) (string, error) {
	message, err := buffer.ReadBytes('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return string(message), nil
}
