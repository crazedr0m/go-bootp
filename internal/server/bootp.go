package server

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

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

// AllocationType тип выделения IP адреса
type AllocationType int

const (
	StaticAllocation  AllocationType = iota // Статическое назначение
	DynamicAllocation                       // Динамическое назначение
)

// AllocatedIP хранит информацию о выделенном IP адресе
type AllocatedIP struct {
	IP      uint32         // IP адрес в виде целого числа
	MAC     string         // MAC адрес клиента
	Subnet  *config.Subnet // Подсеть
	Type    AllocationType // Тип выделения
	Active  bool           // Флаг активности (для статических адресов)
	Expires time.Time      // Время истечения аренды (для динамических адресов)
}

// BOOTPServer представляет BOOTP сервер
type BOOTPServer struct {
	config       *config.DHCPConfig
	conn         *net.UDPConn
	allocatedIP  map[uint32]*AllocatedIP // Выделенные IP адреса (ключ - IP адрес в виде числа)
	allocatedMAC map[string]*AllocatedIP // Выделенные IP адреса (ключ - MAC адрес)
	mutex        sync.Mutex              // Мьютекс для синхронизации доступа к allocated
}

// NewBOOTPServer создает новый BOOTP сервер
func NewBOOTPServer(cfg *config.DHCPConfig) (*BOOTPServer, error) {
	server := &BOOTPServer{
		config:       cfg,
		allocatedIP:  make(map[uint32]*AllocatedIP),
		allocatedMAC: make(map[string]*AllocatedIP),
	}

	// Инициализируем статические назначения
	server.initStaticAllocations()

	return server, nil
}

// initStaticAllocations инициализирует статические назначения IP адресов
func (s *BOOTPServer) initStaticAllocations() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Обрабатываем статические назначения в подсетях
	for _, subnet := range s.config.Subnets {
		for _, host := range subnet.Hosts {
			if host.FixedIP != "" && host.Hardware != "" {
				ip := net.ParseIP(host.FixedIP)
				if ip != nil {
					ipInt := ipToInt(ip)
					mac := strings.ToLower(host.Hardware)
					allocated := &AllocatedIP{
						IP:      ipInt,
						MAC:     mac,
						Subnet:  &subnet,
						Type:    StaticAllocation,
						Active:  false,       // Будет активирован при первом запросе
						Expires: time.Time{}, // Не истекает для статических адресов
					}
					s.allocatedIP[ipInt] = allocated
					s.allocatedMAC[mac] = allocated
				}
			}
		}
	}

	// Обрабатываем глобальные хосты
	for _, host := range s.config.Hosts {
		if host.FixedIP != "" && host.Hardware != "" {
			ip := net.ParseIP(host.FixedIP)
			if ip != nil {
				ipInt := ipToInt(ip)
				mac := strings.ToLower(host.Hardware)
				allocated := &AllocatedIP{
					IP:      ipInt,
					MAC:     mac,
					Subnet:  nil,
					Type:    StaticAllocation,
					Active:  false,       // Будет активирован при первом запросе
					Expires: time.Time{}, // Не истекает для статических адресов
				}
				s.allocatedIP[ipInt] = allocated
				s.allocatedMAC[mac] = allocated
			}
		}
	}
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
	macAddr = strings.ToLower(macAddr)

	// Проверяем статические назначения
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if allocated, exists := s.allocatedMAC[macAddr]; exists && allocated.Type == StaticAllocation {
		// Активируем статический адрес
		allocated.Active = true
		return intToIP(allocated.IP).String(), allocated.Subnet
	}

	// Проверяем динамические назначения
	if allocated, exists := s.allocatedMAC[macAddr]; exists && allocated.Type == DynamicAllocation {
		// Проверяем, не истек ли срок действия
		if allocated.Expires.IsZero() || allocated.Expires.After(time.Now()) {
			// Продлеваем аренду
			allocated.Expires = time.Now().Add(1 * time.Hour)
			return intToIP(allocated.IP).String(), allocated.Subnet
		}
		// Если срок истек, удаляем запись
		delete(s.allocatedIP, allocated.IP)
		delete(s.allocatedMAC, macAddr)
	}

	// Реализовать динамическое назначение IP адресов
	return s.allocateDynamicIP(macAddr)
}

// allocateDynamicIP выделяет динамический IP адрес для клиента
func (s *BOOTPServer) allocateDynamicIP(macAddr string) (string, *config.Subnet) {
	macAddr = strings.ToLower(macAddr)

	// Ищем свободный IP адрес в подсетях с диапазонами
	for _, subnet := range s.config.Subnets {
		if subnet.RangeStart != "" && subnet.RangeEnd != "" {
			startIP := net.ParseIP(subnet.RangeStart)
			endIP := net.ParseIP(subnet.RangeEnd)

			if startIP != nil && endIP != nil {
				// Ищем первый свободный IP в диапазоне
				for ip := ipToInt(startIP); ip <= ipToInt(endIP); ip++ {
					// Проверяем, не занят ли этот IP
					if !s.isIPAllocated(ip) {
						// Найден свободный IP, выделяем его
						allocated := &AllocatedIP{
							IP:      ip,
							MAC:     macAddr,
							Subnet:  &subnet,
							Type:    DynamicAllocation,
							Active:  true,
							Expires: time.Now().Add(1 * time.Hour), // 1 час аренды
						}
						s.allocatedIP[ip] = allocated
						s.allocatedMAC[macAddr] = allocated
						return intToIP(ip).String(), &subnet
					}
				}
			}
		}
	}

	// Не найдено свободных IP адресов
	return "", nil
}

// isIPAllocated проверяет, занят ли IP адрес
func (s *BOOTPServer) isIPAllocated(ip uint32) bool {
	if allocated, exists := s.allocatedIP[ip]; exists {
		// Для статических адресов проверяем активность
		if allocated.Type == StaticAllocation {
			return allocated.Active
		}
		// Для динамических адресов проверяем срок аренды
		if !allocated.Expires.IsZero() && allocated.Expires.Before(time.Now()) {
			// Срок аренды истек, удаляем запись
			delete(s.allocatedIP, ip)
			delete(s.allocatedMAC, allocated.MAC)
			return false
		}
		return true
	}
	return false
}

// Вспомогательные функции для работы с IP адресами
func ipToInt(ip net.IP) uint32 {
	ip = ip.To4()
	return uint32(ip[0])<<24 + uint32(ip[1])<<16 + uint32(ip[2])<<8 + uint32(ip[3])
}

func intToIP(n uint32) net.IP {
	return net.IPv4(byte(n>>24), byte(n>>16), byte(n>>8), byte(n))
}
