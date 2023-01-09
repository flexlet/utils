package utils

import (
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

const (
	MODE_PERM_RW fs.FileMode = 0600
	MODE_PERM_RO fs.FileMode = 0400
)

const (
	STATUS_UP      string = "up"
	STATUS_DOWN    string = "down"
	STATUS_PENDING string = "pending"
)

func ExecCommand(cmd string) (int, error) {
	proc := exec.Command("bash", "-c", cmd)
	err := proc.Run()
	code := proc.ProcessState.ExitCode()
	LogPrintf(LOG_DEBUG, "ExecCommand", "[%d] <-- %s", code, cmd)
	if err != nil {
		return code, err
	}
	return code, nil
}

func MkdirIfNotExist(path string) error {
	d, err := os.Stat(path)
	if err != nil && os.IsNotExist(err) {
		err1 := os.MkdirAll(path, MODE_PERM_RW)
		if err1 != nil {
			return fmt.Errorf("mkdir %s failed: %s", path, err1.Error())
		}
	} else if !d.IsDir() {
		return fmt.Errorf("%s already exist, but not a directory", path)
	}
	return nil
}

func FileExist(f string) bool {
	_, err := os.Stat(f)
	if err != nil && os.IsNotExist(err) {
		return false
	}
	return true
}

func GetProcStatus(pidFile string) string {
	if !FileExist(pidFile) {
		return STATUS_DOWN
	}
	data, err := ioutil.ReadFile(pidFile)
	if err != nil {
		LogPrintf(LOG_ERROR, "GetProcStatus", "read file '%s' failed: %s\n", pidFile, err.Error())
		return STATUS_PENDING
	}
	line := strings.TrimSuffix(string(data), "\n")
	pid, err := strconv.ParseUint(line, 10, 32)
	if err != nil {
		LogPrintf(LOG_ERROR, "GetProcStatus", "parse pid '%s' failed: %s\n", line, err.Error())
		return STATUS_PENDING
	}
	if FileExist(fmt.Sprintf("/proc/%d", pid)) {
		return STATUS_UP
	}
	return STATUS_DOWN
}

func GetInterfaceStatus(dev *string) string {
	cmd := fmt.Sprintf("DEV=%s;exit $(ip a s ${DEV} | grep \"${DEV}.*state UP\" | wc -l)", *dev)
	if cnt, _ := ExecCommand(cmd); cnt == 0 {
		return STATUS_DOWN
	} else {
		return STATUS_UP
	}
}

func GetIPStatus(ipaddress string, prefix *uint8, dev *string) string {
	if prefix != nil {
		ipaddress = ipaddress + fmt.Sprintf("/%d", *prefix)
	}

	var cmd string
	if dev != nil {
		cmd = fmt.Sprintf("exit $(ip a s %s | grep %s | wc -l)", *dev, ipaddress)
	} else {
		cmd = fmt.Sprintf("exit $(ip a s | grep %s | wc -l)", ipaddress)
	}

	if cnt, _ := ExecCommand(cmd); cnt == 0 {
		return STATUS_DOWN
	} else {
		return STATUS_UP
	}
}

func GetInterfaceOfIP(ipaddress string) (*string, *int, error) {
	eths, err := net.Interfaces()
	if err != nil {
		return nil, nil, err
	}
	for i := 0; i < len(eths); i++ {
		eth := eths[i]
		if addrs, err := eth.Addrs(); err != nil { // get addresses
			return nil, nil, err
		} else {
			for j := 0; j < len(addrs); j++ {
				ip := strings.Split(addrs[j].String(), "/")
				if ipaddress == ip[0] {
					prefix, err := strconv.Atoi(ip[1])
					if err != nil {
						return nil, nil, err
					}
					return &eth.Name, &prefix, nil
				}
			}
		}
	}
	return nil, nil, fmt.Errorf("not found interface of '%s'", ipaddress)
}

func DelFileIfExist(f string) error {
	if !FileExist(f) {
		return nil
	}
	return os.Remove(f)
}

func CreateFile(target string, data ...string) error {
	if t, err := os.Create(target); err != nil {
		return err
	} else {
		defer t.Close()
		for _, d := range data {
			if _, err := t.Write([]byte(d)); err != nil {
				return err
			}
		}
	}
	return nil
}

func WriteFile(target string, flag int, mode os.FileMode, data ...string) error {
	if t, err := os.OpenFile(target, flag, mode); err != nil {
		return err
	} else {
		defer t.Close()
		for _, d := range data {
			if _, err := t.Write([]byte(d)); err != nil {
				return err
			}
		}
	}
	return nil
}

func MergeFiles(target string, files ...string) error {
	t, err := os.Create(target)
	if err != nil {
		return err
	}
	defer t.Close()

	for _, source := range files {
		s, err := os.Open(source)

		if err != nil {
			return err
		}

		buf := make([]byte, 1024)
		for {
			len1, err1 := s.Read(buf)
			if err1 != nil && err1 != io.EOF {
				s.Close()
				return err1
			}
			if len1 == 0 {
				break
			}
			if len2, err2 := t.Write(buf[:len1]); err2 != nil || len1 != len2 {
				s.Close()
				return err2
			}
		}

		s.Close()
	}
	return nil
}
