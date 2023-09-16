// package main

// import (
// 	"fmt"
// 	"net"
// 	"os"
// 	"strconv"

// 	binarypack "github.com/canhlinh/go-binary-pack"
// )

// func CreateHeader(command int, commandString []byte, sessionID int, replyID int) ([]byte, error) {
// 	buf, err := newBP().Pack([]string{"H", "H", "H", "H"}, []interface{}{command, 0, sessionID, replyID})
// 	if err != nil {
// 		return nil, err
// 	}
// 	buf = append(buf, commandString...)

// 	unpackPad := []string{
// 		"B", "B", "B", "B", "B", "B", "B", "B",
// 	}

// 	for i := 0; i < len(commandString); i++ {
// 		unpackPad = append(unpackPad, "B")
// 	}

// 	unpackBuf, err := newBP().UnPack(unpackPad, buf)
// 	if err != nil {
// 		return nil, err
// 	}

// 	checksumBuf, err := createCheckSum(unpackBuf)
// 	if err != nil {
// 		return nil, err
// 	}

// 	c, err := newBP().UnPack([]string{"H"}, checksumBuf)
// 	if err != nil {
// 		return nil, err
// 	}
// 	checksum := c[0].(int)

// 	replyID++
// 	if replyID >= 65535 {
// 		replyID -= 65535
// 	}

// 	packData, err := newBP().Pack([]string{"H", "H", "H", "H"}, []interface{}{command, checksum, sessionID, replyID})
// 	if err != nil {
// 		return nil, err
// 	}

// 	return append(packData, commandString...), nil
// }
// func newBP() *binarypack.BinaryPack {
// 	return &binarypack.BinaryPack{}
// }
// func createCheckSum(p []interface{}) ([]byte, error) {
// 	l := len(p)
// 	checksum := 0

// 	for l > 1 {
// 		pack, err := newBP().Pack([]string{"B", "B"}, []interface{}{p[0], p[1]})
// 		if err != nil {
// 			return nil, err
// 		}

// 		unpack, err := newBP().UnPack([]string{"H"}, pack)
// 		if err != nil {
// 			return nil, err
// 		}

// 		c := unpack[0].(int)
// 		checksum += c
// 		p = p[2:]

// 		if checksum > 65535 {
// 			checksum -= 65535
// 		}
// 		l -= 2
// 	}

// 	if l > 0 {
// 		checksum = checksum + p[len(p)-1].(int)
// 	}

// 	for checksum > 65535 {
// 		checksum -= 65535
// 	}

// 	checksum = ^checksum
// 	for checksum < 0 {
// 		checksum += 65535
// 	}

// 	return newBP().Pack([]string{"H"}, []interface{}{checksum})
// }
// func main() {
// 	serverIP := "192.168.50.32" // Replace with the server's IP address
// 	serverPort := 4370          // Replace with the server's port
// 	var sessionID, replyID int
// 	// Connect to the UDP server
// 	conn, err := ConnectUDP(serverIP, serverPort)
// 	if err != nil {
// 		fmt.Println("Error connecting:", err)
// 		os.Exit(1)
// 	}
// 	defer conn.Close()

// 	// Define your custom header command as a string or int
// 	// commandString := 65535 // Replace with your command
// 	// commandBytes := []byte(strconv.Itoa(65535))
// 	commandString := []byte(strconv.Itoa(65535))
// 	header, err := CreateHeader(1000, commandString, sessionID, replyID)
// 	// Send the command to the server
// 	if err := SendCommand(conn, 1000, header); err != nil {
// 		fmt.Println("Error sending command:", err)
// 		os.Exit(1)
// 	}

// 	// Receive and print the response from the server
// 	response, err := ReceiveResponse(conn)
// 	if err != nil {
// 		fmt.Println("Error receiving response:", err)
// 		os.Exit(1)
// 	}

// 	fmt.Println("Server Response:", response)
// }

// // ConnectUDP establishes a UDP connection and returns a UDPConn and an error.
// func ConnectUDP(serverIP string, serverPort int) (*net.UDPConn, error) {
// 	serverAddr := &net.UDPAddr{
// 		IP:   net.ParseIP(serverIP),
// 		Port: serverPort,
// 	}

// 	conn, err := net.DialUDP("udp", nil, serverAddr)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return conn, nil
// }

// // SendCommand sends a command to the server over a UDP connection.
// func SendCommand(conn *net.UDPConn, inCom int, command []byte) error {
// 	_, err := conn.Write(command)
// 	return err
// }

// // ReceiveResponse receives a response from the server over a UDP connection and returns it as a string.
//
//	func ReceiveResponse(conn *net.UDPConn) (string, error) {
//		buffer := make([]byte, 64)
//		n, _, err := conn.ReadFromUDP(buffer)
//		if err != nil {
//			return "", err
//		}
//		return string(buffer[:n]), nil
//	}
package zklib

import (
	"fmt"
	"net"
	"time"
)

const (
	CMD_CONNECT = 1000
	CMD_EXIT    = 1001
	CMD_ACK_OK  = 2000
	USHRT_MAX   = 65535
)

type ZKLib struct {
	IP        string
	Port      int
	Inport    int
	Socket    *net.UDPConn
	ReplyID   int
	DataRecv  []byte
	SessionID int
}

func NewZKLib(options Options) *ZKLib {
	return &ZKLib{
		IP:        options.IP,
		Port:      options.Port,
		Inport:    options.Inport,
		Socket:    nil,
		ReplyID:   -1 + USHRT_MAX,
		DataRecv:  nil,
		SessionID: 0,
	}
}

type Options struct {
	IP     string
	Port   int
	Inport int
}

func (zk *ZKLib) Connect(cb func(error, []byte)) {
	if err := zk.executeCmd(CMD_CONNECT, "", cb); err != nil {
		cb(err, nil)
	}
}

func (zk *ZKLib) Disconnect(cb func(error, []byte)) {
	if err := zk.executeCmd(CMD_EXIT, "", cb); err != nil {
		cb(err, nil)
	}
}

func (zk *ZKLib) executeCmd(command int, commandString string, cb func(error, []byte)) error {
	if command == CMD_CONNECT {
		zk.ReplyID = -1 + USHRT_MAX
	}

	buf := zk.createHeader(command, 0, zk.SessionID, zk.ReplyID, commandString)

	udpAddr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%d", zk.IP, zk.Inport))
	if err != nil {
		return err
	}

	conn, err := net.DialUDP("udp4", nil, udpAddr)
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = conn.Write(buf)
	if err != nil {
		return err
	}

	reply := make([]byte, 1024)
	n, err := conn.Read(reply)
	if err != nil {
		return err
	}

	zk.DataRecv = reply[:n]

	if zk.checkValid() {
		zk.SessionID = int(zk.DataRecv[4]) | int(zk.DataRecv[5])<<8
		zk.ReplyID = int(zk.DataRecv[6]) | int(zk.DataRecv[7])<<8
	}

	cb(nil, zk.DataRecv)
	return nil
}

func (zk *ZKLib) createHeader(command, chksum, sessionID, replyID int, commandString string) []byte {
	bufCommandString := []byte(commandString)
	buf := make([]byte, 8+len(bufCommandString))

	buf[0] = byte(command)
	buf[1] = byte(command >> 8)
	buf[2] = byte(chksum)
	buf[3] = byte(chksum >> 8)
	buf[4] = byte(sessionID)
	buf[5] = byte(sessionID >> 8)
	buf[6] = byte(replyID)
	buf[7] = byte(replyID >> 8)

	copy(buf[8:], bufCommandString)

	chksum = zk.createChkSum(buf)
	buf[2] = byte(chksum)
	buf[3] = byte(chksum >> 8)

	replyID = (replyID + 1) % USHRT_MAX
	buf[6] = byte(replyID)
	buf[7] = byte(replyID >> 8)

	return buf
}

func (zk *ZKLib) createChkSum(p []byte) int {
	var chksum int

	for i := 0; i < len(p); i += 2 {
		if i == len(p)-1 {
			chksum += int(p[i])
		} else {
			chksum += int(p[i]) | int(p[i+1])<<8
		}
		chksum %= USHRT_MAX
	}

	chksum = USHRT_MAX - chksum - 1

	return chksum
}

func (zk *ZKLib) checkValid() bool {
	command := int(zk.DataRecv[0]) | int(zk.DataRecv[1])<<8
	return command == CMD_ACK_OK
}

func (zk *ZKLib) EncodeTime(t time.Time) int {
	d := (t.Year()%100)*12*31 + (int(t.Month())-1)*31 + t.Day() - 1
	d = (d*24*60*60 + t.Hour()*60*60 + t.Minute()*60 + t.Second())

	return d
}

func (zk *ZKLib) DecodeTime(t int) time.Time {
	second := t % 60
	t = (t - second) / 60

	minute := t % 60
	t = (t - minute) / 60

	hour := t % 24
	t = (t - hour) / 24

	day := t%31 + 1
	t = (t - (day - 1)) / 31

	month := t % 12
	t = (t - month) / 12

	year := t + 2000

	return time.Date(year, time.Month(month+1), day, hour, minute, second, 0, time.UTC)
}
