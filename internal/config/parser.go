package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// DHCPConfig представляет конфигурацию ISC-DHCP
type DHCPConfig struct {
	Subnets       []Subnet
	Hosts         []Host
	GlobalOptions map[string]string
}

// Subnet представляет подсеть в конфигурации
type Subnet struct {
	Network    string
	Netmask    string
	RangeStart string
	RangeEnd   string
	Options    map[string]string
	Hosts      []Host
}

// Host представляет хост в конфигурации
type Host struct {
	Name     string
	Hardware string
	Address  string
	FixedIP  string
	Options  map[string]string
}

// ParseConfig парсит конфигурационный файл ISC-DHCP
func ParseConfig(filename string) (*DHCPConfig, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	config := &DHCPConfig{
		Subnets:       make([]Subnet, 0),
		Hosts:         make([]Host, 0),
		GlobalOptions: make(map[string]string),
	}

	// Состояния парсера
	const (
		StateGlobal = iota
		StateSubnet
		StateHostInSubnet
		StateHostGlobal
	)

	state := StateGlobal
	currentSubnet := Subnet{}
	currentHost := Host{}

	scanner := bufio.NewScanner(file)
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())

		// Пропускаем пустые строки и комментарии
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Убираем точку с запятой в конце для обработки
		trimmedLine := strings.TrimSuffix(line, ";")

		// Отладочный вывод
		fmt.Printf("Line %d: State=%d, Line='%s'\n", lineNumber, state, line)

		switch state {
		case StateGlobal:
			// Проверяем начало подсети с учетом пробелов перед {
			if strings.HasPrefix(line, "subnet ") && strings.Contains(line, "{") {
				// Начало подсети
				fmt.Printf("  -> Starting subnet block\n")
				state = StateSubnet
				currentSubnet = Subnet{
					Options: make(map[string]string),
					Hosts:   make([]Host, 0),
				}

				// Убираем { и все после нее, затем убираем концевые пробелы
				blockStart := strings.Index(line, "{")
				if blockStart > 0 {
					subnetDecl := strings.TrimSpace(line[:blockStart])
					// Парсим параметры подсети
					parts := strings.Fields(subnetDecl)
					fmt.Printf("  -> Subnet parts: %v (len=%d)\n", parts, len(parts))
					// parts = [subnet 192.168.1.0 netmask 255.255.255.0]
					// indices: 0      1            2       3
					if len(parts) == 4 && parts[2] == "netmask" {
						currentSubnet.Network = parts[1] // IP адрес сети
						currentSubnet.Netmask = parts[3] // Маска подсети
						fmt.Printf("  -> Network: %s, Netmask: %s\n", currentSubnet.Network, currentSubnet.Netmask)
					}
				}
			} else if strings.HasPrefix(line, "host ") && strings.Contains(line, "{") {
				// Начало глобального хоста
				fmt.Printf("  -> Starting global host block\n")
				state = StateHostGlobal
				// Убираем { и все после нее, затем убираем концевые пробелы
				blockStart := strings.Index(line, "{")
				if blockStart > 0 {
					hostDecl := strings.TrimSpace(line[:blockStart])
					parts := strings.Fields(hostDecl)
					fmt.Printf("  -> Host parts: %v (len=%d)\n", parts, len(parts))
					if len(parts) >= 2 {
						currentHost = Host{
							Name:    parts[1],
							Options: make(map[string]string),
						}
						fmt.Printf("  -> Host name: %s\n", currentHost.Name)
					}
				}
			} else if strings.Contains(line, " ") && !strings.Contains(line, "{") && strings.HasSuffix(line, ";") {
				// Глобальная опция
				fmt.Printf("  -> Processing global option with value\n")
				parts := strings.SplitN(trimmedLine, " ", 2)
				fmt.Printf("  -> Global option parts: %v (len=%d)\n", parts, len(parts))
				if len(parts) == 2 {
					config.GlobalOptions[parts[0]] = parts[1]
					fmt.Printf("  -> Global option: %s = %s\n", parts[0], parts[1])
				}
			} else if strings.HasSuffix(line, ";") && !strings.Contains(line, " ") {
				// Глобальная опция без значения (например, authoritative;)
				fmt.Printf("  -> Processing global option without value\n")
				config.GlobalOptions[trimmedLine] = ""
				fmt.Printf("  -> Global option: %s = ''\n", trimmedLine)
			}

		case StateSubnet:
			if strings.HasPrefix(line, "}") {
				// Конец подсети
				fmt.Printf("  -> Ending subnet block\n")
				config.Subnets = append(config.Subnets, currentSubnet)
				state = StateGlobal
			} else if strings.HasPrefix(line, "host ") && strings.Contains(line, "{") {
				// Начало хоста в подсети
				fmt.Printf("  -> Starting host in subnet block\n")
				state = StateHostInSubnet
				// Убираем { и все после нее, затем убираем концевые пробелы
				blockStart := strings.Index(line, "{")
				if blockStart > 0 {
					hostDecl := strings.TrimSpace(line[:blockStart])
					parts := strings.Fields(hostDecl)
					fmt.Printf("  -> Host parts: %v (len=%d)\n", parts, len(parts))
					if len(parts) >= 2 {
						currentHost = Host{
							Name:    parts[1],
							Options: make(map[string]string),
						}
						fmt.Printf("  -> Host name: %s\n", currentHost.Name)
					}
				}
			} else if strings.HasPrefix(trimmedLine, "range ") {
				// Диапазон IP адресов
				fmt.Printf("  -> Processing range\n")
				parts := strings.Fields(trimmedLine[6:]) // Убираем "range "
				fmt.Printf("  -> Range parts: %v (len=%d)\n", parts, len(parts))
				if len(parts) >= 2 {
					currentSubnet.RangeStart = parts[0]
					currentSubnet.RangeEnd = parts[1]
					fmt.Printf("  -> Range: %s - %s\n", currentSubnet.RangeStart, currentSubnet.RangeEnd)
				}
			} else if strings.HasPrefix(trimmedLine, "option ") {
				// Опция подсети
				fmt.Printf("  -> Processing subnet option\n")
				parts := strings.Fields(trimmedLine[7:]) // Убираем "option "
				fmt.Printf("  -> Option parts: %v (len=%d)\n", parts, len(parts))
				if len(parts) >= 2 {
					// Объединяем все части после ключа в значение
					key := parts[0]
					value := strings.Join(parts[1:], " ")
					// Убираем кавычки, если есть
					value = strings.Trim(value, "\"")
					currentSubnet.Options[key] = value
					fmt.Printf("  -> Subnet option: %s = %s\n", key, value)
				}
			}

		case StateHostInSubnet:
			if strings.HasPrefix(line, "}") {
				// Конец хоста в подсети
				fmt.Printf("  -> Ending host in subnet block\n")
				currentSubnet.Hosts = append(currentSubnet.Hosts, currentHost)
				state = StateSubnet
			} else if strings.HasPrefix(trimmedLine, "hardware ethernet ") {
				// MAC адрес
				fmt.Printf("  -> Processing hardware ethernet\n")
				currentHost.Hardware = strings.TrimSpace(trimmedLine[18:]) // Убираем "hardware ethernet "
				fmt.Printf("  -> Hardware: %s\n", currentHost.Hardware)
			} else if strings.HasPrefix(trimmedLine, "fixed-address ") {
				// Фиксированный IP адрес
				fmt.Printf("  -> Processing fixed-address\n")
				currentHost.FixedIP = strings.TrimSpace(trimmedLine[14:]) // Убираем "fixed-address "
				fmt.Printf("  -> Fixed IP: %s\n", currentHost.FixedIP)
			} else if strings.HasPrefix(trimmedLine, "option ") {
				// Опция хоста
				fmt.Printf("  -> Processing host option\n")
				parts := strings.Fields(trimmedLine[7:]) // Убираем "option "
				fmt.Printf("  -> Option parts: %v (len=%d)\n", parts, len(parts))
				if len(parts) >= 2 {
					// Объединяем все части после ключа в значение
					key := parts[0]
					value := strings.Join(parts[1:], " ")
					// Убираем кавычки, если есть
					value = strings.Trim(value, "\"")
					currentHost.Options[key] = value
					fmt.Printf("  -> Host option: %s = %s\n", key, value)
				}
			}

		case StateHostGlobal:
			if strings.HasPrefix(line, "}") {
				// Конец глобального хоста
				fmt.Printf("  -> Ending global host block\n")
				config.Hosts = append(config.Hosts, currentHost)
				state = StateGlobal
			} else if strings.HasPrefix(trimmedLine, "hardware ethernet ") {
				// MAC адрес
				fmt.Printf("  -> Processing hardware ethernet\n")
				currentHost.Hardware = strings.TrimSpace(trimmedLine[18:]) // Убираем "hardware ethernet "
				fmt.Printf("  -> Hardware: %s\n", currentHost.Hardware)
			} else if strings.HasPrefix(trimmedLine, "fixed-address ") {
				// Фиксированный IP адрес
				fmt.Printf("  -> Processing fixed-address\n")
				currentHost.FixedIP = strings.TrimSpace(trimmedLine[14:]) // Убираем "fixed-address "
				fmt.Printf("  -> Fixed IP: %s\n", currentHost.FixedIP)
			} else if strings.HasPrefix(trimmedLine, "option ") {
				// Опция хоста
				fmt.Printf("  -> Processing host option\n")
				parts := strings.Fields(trimmedLine[7:]) // Убираем "option "
				fmt.Printf("  -> Option parts: %v (len=%d)\n", parts, len(parts))
				if len(parts) >= 2 {
					// Объединяем все части после ключа в значение
					key := parts[0]
					value := strings.Join(parts[1:], " ")
					// Убираем кавычки, если есть
					value = strings.Trim(value, "\"")
					currentHost.Options[key] = value
					fmt.Printf("  -> Host option: %s = %s\n", key, value)
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	fmt.Printf("Parsing complete. Subnets: %d, Hosts: %d, Global options: %d\n",
		len(config.Subnets), len(config.Hosts), len(config.GlobalOptions))

	return config, nil
}
