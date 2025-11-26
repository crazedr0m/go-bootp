# GO-BOOTP Сервер

Полнофункциональный BOOTP/DHCP сервер, написанный на Go, с поддержкой конфигурации ISC-DHCP.

## Возможности

- Полная поддержка BOOTP и DHCP протоколов
- Совместимость с конфигурацией ISC-DHCP
- Поддержка IPv4
- Поддержка TFTP для PXE загрузки
- Поддержка статических назначений и резервирований
- Все опции DHCP (DNS, шлюзы, маски подсетей и т.д.)

## Структура проекта

```
go-bootp/
├── cmd/
│   └── bootpd/
│       └── main.go
├── internal/
│   ├── config/
│   │   └── parser.go
│   ├── server/
│   │   ├── bootp.go
│   │   ├── dhcp.go
│   │   └── tftp.go
│   └── utils/
│       └── helpers.go
├── configs/
│   └── dhcpd.conf
├── go.mod
├── go.sum
└── README.md
```

## Установка

```bash
go mod tidy
go build -o bootpd cmd/bootpd/main.go
```

## Использование

```bash
./bootpd -config /path/to/dhcpd.conf
```

## Конфигурация

Сервер поддерживает стандартный формат конфигурации ISC-DHCP:

```
subnet 192.168.1.0 netmask 255.255.255.0 {
  range 192.168.1.100 192.168.1.200;
  option routers 192.168.1.1;
  option domain-name-servers 8.8.8.8, 8.8.4.4;
  option bootfile-name "pxelinux.0";
  next-server 192.168.1.10;
  
  host client1 {
    hardware ethernet 00:11:22:33:44:55;
    fixed-address 192.168.1.10;
  }
}