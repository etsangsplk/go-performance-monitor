package services

import (
	"github.com/koloo91/performance_monitor/models"
	"fmt"
	"github.com/koloo91/performance_monitor/util"
	"regexp"
	"golang.org/x/crypto/ssh"
	"strconv"
	"time"
	"log"
)

var (
	// CPU
	cpuOldRegex = regexp.MustCompile(`^cpu \s*([0-9]+) ([0-9]+) ([0-9]+) ([0-9]+) ([0-9]+) ([0-9]+) ([0-9]+) ([0-9]+)`)

	// RAM
	memTotalRegex = regexp.MustCompile(`MemTotal:\s*([0-9]+)`)
	buffersRegex  = regexp.MustCompile(`Buffers:\s*([0-9]+)`)
	freeRamRegex  = regexp.MustCompile(`MemFree:\s*([0-9]+)`)
	cachedRegex   = regexp.MustCompile(`Cached:\s*([0-9]+)`)

	// LoadAverage
	loadAverageRegex = regexp.MustCompile(`average[s]?:\s*([0-9.]+)`)

	// Network
	networkRegex = regexp.MustCompile(`\s*([a-z0-9]+):\s*([0-9]+)\s*([0-9]+)\s*([0-9]+)\s*([0-9]+)\s*([0-9]+)\s*([0-9]+)\s*([0-9]+)\s*([0-9]+)\s*([0-9]+)`)

	// Disk
	diskRegex = regexp.MustCompile(`(\d+)\s+(\d+)\s+(\d+)\s+(\d+)%`)
)

func Init(sshConfiguration *models.SshConfiguration) {
	fmt.Println(*sshConfiguration)

	serverWithPort := fmt.Sprintf("%s:%d", sshConfiguration.Server, sshConfiguration.Port)
	client, session, err := util.ConnectToHost(sshConfiguration.UserName, sshConfiguration.Password, serverWithPort)
	defer session.Close()

	if err != nil {
		panic(err)
	}

	serverStatistics := models.ServerStatistics{}

	cpuChannel := make(chan models.CpuData)
	memoryChannel := make(chan models.MemoryData)
	loadAverageChannel := make(chan models.LoadAverage)
	networkChannel := make(chan models.NetworkData)
	diskChannel := make(chan models.DiskData)

	ticker := time.NewTicker(time.Second)
	go func() {
		for range ticker.C {
			getCpuData(client, cpuChannel)
			getMemoryData(client, memoryChannel)
			getLoadAverage(client, loadAverageChannel)
			getNetwork(client, networkChannel)
			getDiskData(client, diskChannel)
		}
	}()

	for {
		select {
		case cpu := <-cpuChannel:
			if total, idle := cpu.Total, cpu.Idle; total > 0 && idle > 0 {
				idleDelta := idle - serverStatistics.Cpu.Idle
				totalDelta := total - serverStatistics.Cpu.Total

				free := (idleDelta * 100.0) / (totalDelta + 0.5)
				fmt.Println("Free:", 100-free, "%")
			}

			serverStatistics.Cpu = cpu
		case memory := <-memoryChannel:
			fmt.Println(memoryToText(memory))
			serverStatistics.Memory = memory
		case loadAverage := <-loadAverageChannel:
			fmt.Printf("LoadAverage: %v\n", loadAverage.Value)
			serverStatistics.LoadAverage = loadAverage
		case networkData := <-networkChannel:
			fmt.Println(networkData)
			serverStatistics.Network = networkData
		case diskData := <-diskChannel:
			fmt.Println(diskData)
			serverStatistics.Disk = diskData
		default:
		}
	}
}

func getCpuData(client *ssh.Client, resultChannel chan<- models.CpuData) {
	go func() {
		session, err := client.NewSession()
		if err != nil {
			panic(err)
		}
		defer session.Close()

		out, err := session.CombinedOutput("cat /proc/stat")
		if err != nil {
			log.Println("Unable to execute command 'cat /proc/stat'")
			resultChannel <- models.CpuData{}
		}

		if cpuOldRegex.Match(out) {
			for _, value := range cpuOldRegex.FindAllStringSubmatch(string(out), -1) {
				user, _ := strconv.ParseFloat(value[1], 64)
				nice, _ := strconv.ParseFloat(value[2], 64)
				sys, _ := strconv.ParseFloat(value[3], 64)
				idle, _ := strconv.ParseFloat(value[4], 64)
				iowait, _ := strconv.ParseFloat(value[5], 64)
				irq, _ := strconv.ParseFloat(value[6], 64)
				softirq, _ := strconv.ParseFloat(value[7], 64)
				steal, _ := strconv.ParseFloat(value[8], 64)

				total := user + nice + sys + idle + iowait + irq + softirq + steal
				resultChannel <- models.CpuData{total, idle}
			}
		} else {
			fmt.Println("String does not match")
		}
	}()
}

func getMemoryData(client *ssh.Client, resultChannel chan<- models.MemoryData) {
	go func() {
		session, err := client.NewSession()
		if err != nil {
			panic(err)
		}
		defer session.Close()

		out, err := session.CombinedOutput("cat /proc/meminfo")
		if err != nil {
			log.Println("Unable to execute command 'cat /proc/meminfo'")
			resultChannel <- models.MemoryData{}
		}

		memoryData := models.MemoryData{}

		if memTotalRegex.Match(out) {
			for _, value := range memTotalRegex.FindAllStringSubmatch(string(out), -1) {
				if total, err := strconv.ParseFloat(value[1], 64); err == nil {
					memoryData.Total = total
				}
			}
		} else {
			fmt.Println("String does not match for 'buffers'")
		}

		if buffersRegex.Match(out) {
			for _, value := range buffersRegex.FindAllStringSubmatch(string(out), -1) {
				if buffers, err := strconv.ParseFloat(value[1], 64); err == nil {
					memoryData.Bufferes = buffers
				}
			}
		} else {
			fmt.Println("String does not match for 'buffers'")
		}

		if freeRamRegex.Match(out) {
			for _, value := range freeRamRegex.FindAllStringSubmatch(string(out), -1) {
				if free, err := strconv.ParseFloat(value[1], 64); err == nil {
					memoryData.Free = free
				}
			}
		} else {
			fmt.Println("String does not match for 'free'")
		}

		if cachedRegex.Match(out) {
			for _, value := range cachedRegex.FindAllStringSubmatch(string(out), -1) {
				if cached, err := strconv.ParseFloat(value[1], 64); err == nil {
					memoryData.Cached = cached
				}
			}
		} else {
			fmt.Println("String does not match for 'chached'")
		}

		resultChannel <- memoryData
	}()
}

func getLoadAverage(client *ssh.Client, resultChannel chan<- models.LoadAverage) {

	go func() {
		session, err := client.NewSession()
		if err != nil {
			panic(err)
		}
		defer session.Close()

		out, err := session.CombinedOutput("uptime")
		if err != nil {
			log.Println("Unable to execute command 'uptime'")
			resultChannel <- models.LoadAverage{}
		}

		if loadAverageRegex.Match(out) {
			for _, value := range loadAverageRegex.FindAllStringSubmatch(string(out), -1) {
				value, _ := strconv.ParseFloat(value[1], 64)
				resultChannel <- models.LoadAverage{value}
			}
		} else {
			fmt.Printf("String does not match %v\n", loadAverageRegex)
			resultChannel <- models.LoadAverage{}
		}
	}()
}

func getNetwork(client *ssh.Client, resultChannel chan<- models.NetworkData) {

	go func() {
		session, err := client.NewSession()
		if err != nil {
			panic(err)
		}
		defer session.Close()

		result := models.NetworkData{}
		result.InterfaceData = make([]models.NetworkInterfaceData, 0)

		out, err := session.CombinedOutput("cat /proc/net/dev")
		if err != nil {
			log.Println("Unable to execute command 'cat /proc/net/dev'")
			resultChannel <- result
		}

		if networkRegex.Match(out) {
			for _, value := range networkRegex.FindAllStringSubmatch(string(out), -1) {
				name := value[1]
				rx, _ := strconv.ParseFloat(value[2], 64)
				tx, _ := strconv.ParseFloat(value[10], 64)

				interfaceData := models.NetworkInterfaceData{Name: name, Rx: rx, Tx: tx}
				result.InterfaceData = append(result.InterfaceData, interfaceData)
			}
		} else {
			fmt.Printf("String does not match %v\n", networkRegex)
		}

		result.LastUpdate = time.Now()
		resultChannel <- result
	}()
}

func getDiskData(client *ssh.Client, resultChannel chan<- models.DiskData) {

	go func() {
		session, err := client.NewSession()
		if err != nil {
			panic(err)
		}
		defer session.Close()

		out, err := session.CombinedOutput("df | grep ' /$'")
		if err != nil {
			log.Println("Unable to execute command 'df | grep ' /$''")
			resultChannel <- models.DiskData{}
		}

		if diskRegex.Match(out) {
			for _, value := range diskRegex.FindAllStringSubmatch(string(out), -1) {
				size, _ := strconv.ParseFloat(value[1], 64)
				use, _ := strconv.ParseFloat(value[2], 64)
				available, _ := strconv.ParseFloat(value[3], 64)
				useInPercent, _ := strconv.ParseFloat(value[4], 64)

				resultChannel <- models.DiskData{size, use, available, useInPercent}
			}
		} else {
			fmt.Printf("String does not match %v\n", diskRegex)
			resultChannel <- models.DiskData{}
		}
	}()
}

func memoryToText(memoryData models.MemoryData) string {
	total := memoryData.Total - (memoryData.Free + memoryData.Bufferes + memoryData.Cached)

	if total > 1048576 {
		return fmt.Sprintf("%f GB", total/1024/1024)
	} else if total > 1024 {
		return fmt.Sprintf("%f MB", total/1024)
	} else {
		return fmt.Sprintf("%f KB", total)
	}
}
