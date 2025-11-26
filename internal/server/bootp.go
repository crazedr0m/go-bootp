package server

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/user/go-bootp/internal/config"
)

const (
	BOOTPRequest = 1
	BOOTPReply   = 2

	HTYPE_ETHER = 1

	BOOTP_PORT = 67
)

// BOOTPHeader представляет заголовок BOOTP пакета
type BOOTPHeader struct {
	Op     uint8     // Operation Code
	Htype  uint8     // Hardware Type
	Hlen   uint8     // Hardware Address Length
	Hops   uint8     // Hops
	Xid    uint32    // Transaction ID
	Secs   uint16    // Seconds elapsed
	Flags  uint16    // Bootp flags
	Ciaddr [4]byte   // Client IP address
	Yiaddr [4]byte   // Your IP address
	Siaddr [4]byte   // Server IP address
	Giaddr [4]byte   // Gateway IP address
	Chaddr [16]byte  // Client hardware address
	Sname  [64]byte  // Server host name
	File   [128]byte // Boot file name
	Magic  [4]byte   // Magic cookie
}

// BOOTPServer представляет BOOTP сервер
type BOOTPServer struct {
	config *config.DHCPConfig
	conn   *net.UDPConn
}

// NewBOOTPServer создает новый BOOTP сервер
func NewBOOTPServer(cfg *config.DHCPConfig) (*BOOTPServer, error) {
	server := &BOOTPServer{
		config: cfg,
	}

	return server, nil
}

// Start запускает BOOTP сервер
func (s *BOOTPServer) Start() error {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", BOOTP_PORT))
	if err != nil {
		return err
	}

	s.conn, err = net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}

	logrus.Infof("BOOTP server listening on %s", addr.String())

	// Запуск обработки запросов в отдельной горутине
	go s.handleRequests()

	return nil
}

// Stop останавливает BOOTP сервер
func (s *BOOTPServer) Stop() {
	if s.conn != nil {
		s.conn.Close()
	}
}

// handleRequests обрабатывает входящие BOOTP запросы
func (s *BOOTPServer) handleRequests() {
	buffer := make([]byte, 1024)

	for {
		n, clientAddr, err := s.conn.ReadFromUDP(buffer)
		if err != nil {
			logrus.Errorf("Error reading UDP message: %v", err)
			continue
		}

		// Парсим BOOTP заголовок
		header := &BOOTPHeader{}
		reader := bytes.NewReader(buffer[:n])
		err = binary.Read(reader, binary.BigEndian, header)
		if err != nil {
			logrus.Errorf("Error parsing BOOTP header: %v", err)
			continue
		}

		// Обрабатываем только BOOTP запросы
		if header.Op != BOOTPRequest {
			continue
		}

		// Обрабатываем запрос
		reply := s.processRequest(header)

		// Отправляем ответ
		var replyBuffer bytes.Buffer
		err = binary.Write(&replyBuffer, binary.BigEndian, reply)
		if err != nil {
			logrus.Errorf("Error serializing BOOTP reply: %v", err)
			continue
		}

		_, err = s.conn.WriteToUDP(replyBuffer.Bytes(), clientAddr)
		if err != nil {
			logrus.Errorf("Error sending BOOTP reply: %v", err)
		}
	}
}

// processRequest обрабатывает BOOTP запрос и формирует ответ
func (s *BOOTPServer) processRequest(request *BOOTPHeader) *BOOTPHeader {
	reply := &BOOTPHeader{}

	// Копируем поля из запроса
	reply.Op = BOOTPReply
	reply.Htype = request.Htype
	reply.Hlen = request.Hlen
	reply.Hops = 0
	reply.Xid = request.Xid
	reply.Secs = 0
	reply.Flags = request.Flags
	copy(reply.Chaddr[:], request.Chaddr[:])

	// Получаем MAC адрес клиента
	macAddr := fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x",
		request.Chaddr[0], request.Chaddr[1], request.Chaddr[2],
		request.Chaddr[3], request.Chaddr[4], request.Chaddr[5])

	// Ищем конфигурацию для клиента
	clientIP, subnet := s.findClientConfig(macAddr)
	if clientIP == "" {
		logrus.Warnf("No configuration found for client %s", macAddr)
		return nil
	}

	// Устанавливаем IP адреса
	copy(reply.Yiaddr[:], net.ParseIP(clientIP).To4())

	if subnet != nil {
		// Устанавливаем адрес сервера
		if nextServer, ok := subnet.Options["tftp-server-name"]; ok {
			copy(reply.Siaddr[:], net.ParseIP(nextServer).To4())
		}

		// Устанавливаем имя файла загрузки
		if bootfile, ok := subnet.Options["bootfile-name"]; ok {
			copy(reply.File[:], []byte(bootfile))
		}
	}

	// Устанавливаем magic cookie
	reply.Magic = [4]byte{99, 130, 83, 99}

	return reply
}

// findClientConfig находит конфигурацию для клиента по MAC адресу
func (s *BOOTPServer) findClientConfig(macAddr string) (string, *config.Subnet) {
	// Ищем статические назначения в подсетях
	for _, subnet := range s.config.Subnets {
		for _, host := range subnet.Hosts {
			if strings.ToLower(host.Hardware) == strings.ToLower(macAddr) {
				return host.FixedIP, &subnet
			}
		}
	}

	// Ищем глобальные хосты
	for _, host := range s.config.Hosts {
		if strings.ToLower(host.Hardware) == strings.ToLower(macAddr) {
			return host.FixedIP, nil
		}
	}

	// TODO: Реализовать динамическое назначение IP адресов
	return "", nil
}
