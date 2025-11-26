package config

import (
	"os"
	"testing"
)

func TestParseGlobalOptions(t *testing.T) {
	// Создаем тестовую конфигурацию с глобальными опциями
	configContent := `# Пример конфигурации ISC-DHCP для тестирования
default-lease-time 600;
max-lease-time 7200;
log-facility local7;
authoritative;
`

	// Создаем временный файл
	tmpfile, err := os.CreateTemp("", "dhcpd_test.conf")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	// Записываем тестовую конфигурацию в файл
	if _, err := tmpfile.Write([]byte(configContent)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	// Тестируем парсер
	cfg, err := ParseConfig(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	// Проверяем глобальные опции
	if len(cfg.GlobalOptions) != 4 {
		t.Errorf("Expected 4 global options, got %d", len(cfg.GlobalOptions))
	}

	if leaseTime, ok := cfg.GlobalOptions["default-lease-time"]; !ok || leaseTime != "600" {
		t.Errorf("Expected default-lease-time 600, got %s", leaseTime)
	}

	if maxLeaseTime, ok := cfg.GlobalOptions["max-lease-time"]; !ok || maxLeaseTime != "7200" {
		t.Errorf("Expected max-lease-time 7200, got %s", maxLeaseTime)
	}

	if logFacility, ok := cfg.GlobalOptions["log-facility"]; !ok || logFacility != "local7" {
		t.Errorf("Expected log-facility local7, got %s", logFacility)
	}
}

func TestParseSubnet(t *testing.T) {
	// Создаем тестовую конфигурацию с подсетью
	configContent := `subnet 192.168.1.0 netmask 255.255.255.0 {
  range 192.168.1.100 192.168.1.200;
  option routers 192.168.1.1;
  option domain-name-servers 8.8.8.8, 8.8.4.4;
  option domain-name "local.network";
  option bootfile-name "pxelinux.0";
  option tftp-server-name "192.168.1.10";
}`

	// Создаем временный файл
	tmpfile, err := os.CreateTemp("", "dhcpd_test.conf")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	// Записываем тестовую конфигурацию в файл
	if _, err := tmpfile.Write([]byte(configContent)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	// Тестируем парсер
	cfg, err := ParseConfig(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	// Проверяем подсети
	if len(cfg.Subnets) != 1 {
		t.Fatalf("Expected 1 subnet, got %d", len(cfg.Subnets))
	}

	subnet := cfg.Subnets[0]
	if subnet.Network != "192.168.1.0" {
		t.Errorf("Expected network 192.168.1.0, got %s", subnet.Network)
	}

	if subnet.Netmask != "255.255.255.0" {
		t.Errorf("Expected netmask 255.255.255.0, got %s", subnet.Netmask)
	}

	if subnet.RangeStart != "192.168.1.100" {
		t.Errorf("Expected range start 192.168.1.100, got %s", subnet.RangeStart)
	}

	if subnet.RangeEnd != "192.168.1.200" {
		t.Errorf("Expected range end 192.168.1.200, got %s", subnet.RangeEnd)
	}

	// Проверяем опции подсети
	if len(subnet.Options) != 5 {
		t.Errorf("Expected 5 subnet options, got %d", len(subnet.Options))
	}

	if routers, ok := subnet.Options["routers"]; !ok || routers != "192.168.1.1" {
		t.Errorf("Expected routers 192.168.1.1, got %s", routers)
	}

	if dns, ok := subnet.Options["domain-name-servers"]; !ok || dns != "8.8.8.8, 8.8.4.4" {
		t.Errorf("Expected domain-name-servers 8.8.8.8, 8.8.4.4, got %s", dns)
	}

	if domain, ok := subnet.Options["domain-name"]; !ok || domain != "local.network" {
		t.Errorf("Expected domain-name local.network, got %s", domain)
	}

	if bootfile, ok := subnet.Options["bootfile-name"]; !ok || bootfile != "pxelinux.0" {
		t.Errorf("Expected bootfile-name pxelinux.0, got %s", bootfile)
	}

	if tftp, ok := subnet.Options["tftp-server-name"]; !ok || tftp != "192.168.1.10" {
		t.Errorf("Expected tftp-server-name 192.168.1.10, got %s", tftp)
	}
}

func TestParseHostInSubnet(t *testing.T) {
	// Создаем тестовую конфигурацию с хостом в подсети
	configContent := `subnet 192.168.1.0 netmask 255.255.255.0 {
  host client1 {
    hardware ethernet 00:11:22:33:44:55;
    fixed-address 192.168.1.10;
  }
}`

	// Создаем временный файл
	tmpfile, err := os.CreateTemp("", "dhcpd_test.conf")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	// Записываем тестовую конфигурацию в файл
	if _, err := tmpfile.Write([]byte(configContent)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	// Тестируем парсер
	cfg, err := ParseConfig(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	// Проверяем подсети
	if len(cfg.Subnets) != 1 {
		t.Fatalf("Expected 1 subnet, got %d", len(cfg.Subnets))
	}

	subnet := cfg.Subnets[0]

	// Проверяем хосты в подсети
	if len(subnet.Hosts) != 1 {
		t.Fatalf("Expected 1 host in subnet, got %d", len(subnet.Hosts))
	}

	host := subnet.Hosts[0]
	if host.Name != "client1" {
		t.Errorf("Expected host name client1, got %s", host.Name)
	}

	if host.Hardware != "00:11:22:33:44:55" {
		t.Errorf("Expected hardware 00:11:22:33:44:55, got %s", host.Hardware)
	}

	if host.FixedIP != "192.168.1.10" {
		t.Errorf("Expected fixed IP 192.168.1.10, got %s", host.FixedIP)
	}
}

func TestParseGlobalHost(t *testing.T) {
	// Создаем тестовую конфигурацию с глобальным хостом
	configContent := `host global-client {
  hardware ethernet aa:bb:cc:dd:ee:ff;
  fixed-address 192.168.2.10;
}`

	// Создаем временный файл
	tmpfile, err := os.CreateTemp("", "dhcpd_test.conf")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	// Записываем тестовую конфигурацию в файл
	if _, err := tmpfile.Write([]byte(configContent)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	// Тестируем парсер
	cfg, err := ParseConfig(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	// Проверяем глобальные хосты
	if len(cfg.Hosts) != 1 {
		t.Fatalf("Expected 1 global host, got %d", len(cfg.Hosts))
	}

	host := cfg.Hosts[0]
	if host.Name != "global-client" {
		t.Errorf("Expected host name global-client, got %s", host.Name)
	}

	if host.Hardware != "aa:bb:cc:dd:ee:ff" {
		t.Errorf("Expected hardware aa:bb:cc:dd:ee:ff, got %s", host.Hardware)
	}

	if host.FixedIP != "192.168.2.10" {
		t.Errorf("Expected fixed IP 192.168.2.10, got %s", host.FixedIP)
	}
}

func TestParseCompleteConfig(t *testing.T) {
	// Создаем тестовую конфигурацию с полной конфигурацией
	configContent := `# Пример конфигурации ISC-DHCP для тестирования
default-lease-time 600;
max-lease-time 7200;
authoritative;

subnet 192.168.1.0 netmask 255.255.255.0 {
  range 192.168.1.100 192.168.1.200;
  option routers 192.168.1.1;
  option domain-name-servers 8.8.8.8, 8.8.4.4;
  option domain-name "local.network";
  option bootfile-name "pxelinux.0";
  option tftp-server-name "192.168.1.10";
  
  host client1 {
    hardware ethernet 00:11:22:33:44:55;
    fixed-address 192.168.1.10;
  }
}

host global-client {
  hardware ethernet aa:bb:cc:dd:ee:ff;
  fixed-address 192.168.2.10;
}
`

	// Создаем временный файл
	tmpfile, err := os.CreateTemp("", "dhcpd_test.conf")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	// Записываем тестовую конфигурацию в файл
	if _, err := tmpfile.Write([]byte(configContent)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	// Тестируем парсер
	cfg, err := ParseConfig(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	// Проверяем глобальные опции
	if len(cfg.GlobalOptions) != 3 {
		t.Errorf("Expected 3 global options, got %d", len(cfg.GlobalOptions))
	}

	// Проверяем подсети
	if len(cfg.Subnets) != 1 {
		t.Fatalf("Expected 1 subnet, got %d", len(cfg.Subnets))
	}

	subnet := cfg.Subnets[0]
	if subnet.Network != "192.168.1.0" {
		t.Errorf("Expected network 192.168.1.0, got %s", subnet.Network)
	}

	if subnet.Netmask != "255.255.255.0" {
		t.Errorf("Expected netmask 255.255.255.0, got %s", subnet.Netmask)
	}

	// Проверяем хосты в подсети
	if len(subnet.Hosts) != 1 {
		t.Fatalf("Expected 1 host in subnet, got %d", len(subnet.Hosts))
	}

	host := subnet.Hosts[0]
	if host.Name != "client1" {
		t.Errorf("Expected host name client1, got %s", host.Name)
	}

	// Проверяем глобальные хосты
	if len(cfg.Hosts) != 1 {
		t.Fatalf("Expected 1 global host, got %d", len(cfg.Hosts))
	}

	globalHost := cfg.Hosts[0]
	if globalHost.Name != "global-client" {
		t.Errorf("Expected global host name global-client, got %s", globalHost.Name)
	}
}
