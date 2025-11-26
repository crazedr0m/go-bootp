package server

import (
	"bytes"
	"net"
	"testing"

	"github.com/user/go-bootp/internal/config"
)

func TestFindClientConfig(t *testing.T) {
	// Создаем тестовую конфигурацию
	cfg := &config.DHCPConfig{
		Subnets: []config.Subnet{
			{
				Network: "192.168.1.0",
				Netmask: "255.255.255.0",
				Hosts: []config.Host{
					{
						Name:     "client1",
						Hardware: "00:11:22:33:44:55",
						FixedIP:  "192.168.1.10",
					},
				},
			},
		},
		Hosts: []config.Host{
			{
				Name:     "global-client",
				Hardware: "aa:bb:cc:dd:ee:ff",
				FixedIP:  "192.168.2.10",
			},
		},
	}

	// Создаем сервер с тестовой конфигурацией
	server := &BOOTPServer{
		config: cfg,
	}

	// Тестируем поиск клиента в подсети
	ip, subnet := server.findClientConfig("00:11:22:33:44:55")
	if ip != "192.168.1.10" {
		t.Errorf("Expected IP 192.168.1.10, got %s", ip)
	}
	if subnet == nil {
		t.Error("Expected subnet, got nil")
	}

	// Тестируем поиск глобального клиента
	ip, subnet = server.findClientConfig("aa:bb:cc:dd:ee:ff")
	if ip != "192.168.2.10" {
		t.Errorf("Expected IP 192.168.2.10, got %s", ip)
	}
	if subnet != nil {
		t.Error("Expected nil subnet for global host")
	}

	// Тестируем поиск несуществующего клиента
	ip, subnet = server.findClientConfig("00:00:00:00:00:00")
	if ip != "" {
		t.Errorf("Expected empty IP, got %s", ip)
	}
	if subnet != nil {
		t.Error("Expected nil subnet for non-existent host")
	}
}

func TestProcessRequest(t *testing.T) {
	// Создаем тестовую конфигурацию
	cfg := &config.DHCPConfig{
		Subnets: []config.Subnet{
			{
				Network: "192.168.1.0",
				Netmask: "255.255.255.0",
				Options: map[string]string{
					"tftp-server-name": "192.168.1.10",
					"bootfile-name":    "pxelinux.0",
				},
				Hosts: []config.Host{
					{
						Name:     "client1",
						Hardware: "00:11:22:33:44:55",
						FixedIP:  "192.168.1.10",
					},
				},
			},
		},
	}

	// Создаем сервер с тестовой конфигурацией
	server := &BOOTPServer{
		config: cfg,
	}

	// Создаем тестовый BOOTP запрос
	request := &BOOTPHeader{
		Op:     BOOTPRequest,
		Htype:  HTYPE_ETHER,
		Hlen:   6,
		Xid:    0x12345678,
		Chaddr: [16]byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	}

	// Обрабатываем запрос
	reply := server.processRequest(request)

	// Проверяем ответ
	if reply == nil {
		t.Fatal("Expected reply, got nil")
	}

	if reply.Op != BOOTPReply {
		t.Errorf("Expected reply op %d, got %d", BOOTPReply, reply.Op)
	}

	if reply.Htype != HTYPE_ETHER {
		t.Errorf("Expected htype %d, got %d", HTYPE_ETHER, reply.Htype)
	}

	if reply.Hlen != 6 {
		t.Errorf("Expected hlen 6, got %d", reply.Hlen)
	}

	if reply.Xid != 0x12345678 {
		t.Errorf("Expected xid 0x12345678, got 0x%x", reply.Xid)
	}

	// Проверяем IP адрес клиента
	expectedIP := net.ParseIP("192.168.1.10").To4()
	if !bytes.Equal(reply.Yiaddr[:], expectedIP) {
		t.Errorf("Expected yiaddr %v, got %v", expectedIP, reply.Yiaddr[:])
	}

	// Проверяем IP адрес сервера
	expectedServerIP := net.ParseIP("192.168.1.10").To4()
	if !bytes.Equal(reply.Siaddr[:], expectedServerIP) {
		t.Errorf("Expected siaddr %v, got %v", expectedServerIP, reply.Siaddr[:])
	}

	// Проверяем имя файла загрузки
	expectedFile := "pxelinux.0"
	if string(bytes.Trim(reply.File[:], "\x00")) != expectedFile {
		t.Errorf("Expected file %s, got %s", expectedFile, string(reply.File[:]))
	}

	// Проверяем magic cookie
	expectedMagic := [4]byte{99, 130, 83, 99}
	if reply.Magic != expectedMagic {
		t.Errorf("Expected magic %v, got %v", expectedMagic, reply.Magic)
	}
}
