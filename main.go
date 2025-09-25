package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

type header struct {
	header     string
	content    string
	indent     int
	subHeaders []header
}

func startsWith(src string, pattern string) bool {
	if len(src) < len(pattern) {
		return false
	}

	for i, _ := range pattern {
		if src[i] != pattern[i] {
			return false
		}
	}

	return true
}

func getHeader(content string) (header, string) {
	contentSplit := strings.Split(content, "\n")
	var currHeader header = header{}

	if len(contentSplit) < 1 {
		return header{}, ""
	}

	if !strings.HasPrefix(contentSplit[0], "#") {
		panic("no header found")
	} else {
		counter := 0
		for strings.HasPrefix(contentSplit[0], "#") {
			contentSplit[0], _ = strings.CutPrefix(contentSplit[0], "#")
			counter += 1
		}
		contentSplit[0], _ = strings.CutPrefix(contentSplit[0], " ")
		if contentSplit[0] == "" {
			panic("pls add a title to all your headers")
		}
		currHeader.header = contentSplit[0]
		currHeader.indent = counter
	}

	endIdx := len(contentSplit)
	for i := 1; i < len(contentSplit); i++ {
		if strings.HasPrefix(contentSplit[i], "#") {
			endIdx = i
			break
		}
	}
	currHeader.content = strings.Join(contentSplit[1:endIdx], "\n")
	return currHeader, strings.Join(contentSplit[endIdx:], "\n")
}

func getTerminalSize() (int, int, error) {
	file, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()

	var winsize struct {
		Row    uint16
		Col    uint16
		Xpixel uint16
		Ypixel uint16
	}

	_, _, ioctlError := syscall.Syscall(syscall.SYS_IOCTL,
		file.Fd(),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(&winsize)))

	if ioctlError != 0 {
		return 0, 0, ioctlError
	}

	return int(winsize.Row), int(winsize.Col), nil
}

func main() {
	path := os.Args[len(os.Args)-1]
	content, err := os.ReadFile(path)
	if err != nil {
		fmt.Println("cannot open file")
	}
	file := []header{}
	var tempHeader header = header{}
	ret := string(content)
	// ret := string(content)[:len(string(content))-1]
	//file tree
	for ret != "" {
		tempHeader, ret = getHeader(ret)
		if len(file) == 0 {
			if tempHeader.indent != 1 {
				panic("pls start with first header")
			} else {
				file = append(file, tempHeader)
			}
			continue
		}

		if tempHeader.indent == 1 {
			file = append(file, tempHeader)
			continue
		}

		var tempObj *header = &file[len(file)-1]
		for i := 2; i < tempHeader.indent; i++ {
			if len(tempObj.subHeaders) < 1 {
				panic("indentation error")
			}
			tempObj = &tempObj.subHeaders[len(tempObj.subHeaders)-1]
		}
		tempObj.subHeaders = append(tempObj.subHeaders, tempHeader)
	}

	presiList := flatten(file, "")
	for i := 0; i < len(presiList); i++ {
		printFormated(presiList[i])

		terminalHeight, terminalWidth, err := getTerminalSize()
		if err != nil {
			fmt.Println("Error getting terminal size:", err)
			terminalHeight = 24 // Default height
			terminalWidth = 80  // Default width
		}

		status := strconv.Itoa(i+1) + "/" + strconv.Itoa(len(presiList)) + " : " + path
		fmt.Printf("\033[%d;%dH", terminalHeight, terminalWidth-len(status)+1)
		fmt.Println(status)

		exec.Command("stty", "-F", "/dev/tty", "cbreak", "min", "1").Run()
		exec.Command("stty", "-F", "/dev/tty", "-echo").Run()

		var b []byte = make([]byte, 1)
		os.Stdin.Read(b)
		if b[0] == 127 && i > 0 {
			i -= 2
		} else if b[0] == 127 {
			i -= 1
		}
	}
}

type printConfig struct {
	bold   bool
	italic bool
	code   bool
}

func printFormated(headerIn header) {
	fmt.Print("\033[H\033[2J")
	fmt.Println("")
	_, terminalWidth, err := getTerminalSize()
	offSetX := 0
	offSetY := 2
	if err == nil && terminalWidth >= len(headerIn.header) {
		offSetX = (terminalWidth - len(headerIn.header) + 2) / 2
	}
	fmt.Printf("\033[%d;%dH", offSetY, offSetX)
	fmt.Print("╭")
	for i := 0; i < len(headerIn.header); i++ {
		fmt.Print("─")
	}
	fmt.Println("╮")
	fmt.Printf("\033[%d;%dH", offSetY+1, offSetX)
	fmt.Println("│" + headerIn.header + "│")
	fmt.Printf("\033[%d;%dH", offSetY+2, offSetX)
	fmt.Print("╰")
	for i := 0; i < len(headerIn.header); i++ {
		fmt.Print("─")
	}
	fmt.Println("╯")

	strSplit := strings.Split(headerIn.content, "\n")
	defaultPrint := printConfig{false, false, false}

	for _, l := range strSplit {
		if strings.HasPrefix(l, "```") {
			defaultPrint.code = !defaultPrint.code
			continue
		}
		if defaultPrint.code {
			printChar(l, defaultPrint)
		}

		if strings.HasPrefix(l, "---") {
			fmt.Println("──────────────────────────────")
			continue
		}
		for strings.HasPrefix(l, " ") {
			fmt.Print("\t")
			l = l[1:]
		}

		if strings.HasPrefix(l, "-") {
			printChar("•", defaultPrint)
			l = l[1:]
		}

		m := []rune(l)
		for i := 0; i < len(m); i++ {
			if strings.HasPrefix(string(m[i:]), "\\") && i < len(m)-1 {
				printChar(string(m[i+1]), defaultPrint)
				i++
				continue
			}
			i += printFor(m[i:], "__", printConfig{true, false, false})
			i += printFor(m[i:], "**", printConfig{true, false, false})
			i += printFor(m[i:], "*", printConfig{false, true, false})
			i += printFor(m[i:], "_", printConfig{false, true, false})
			i += printFor(m[i:], "`", printConfig{false, false, true})

			if i < len(m) {
				printChar(string(m[i]), defaultPrint)
			}

		}
		fmt.Print("\n")
	}
}

func printFor(stringIn []rune, char string, printIn printConfig) int {
	i := 0
	if len(stringIn) < 1 {
		return 0
	}
	if strings.HasPrefix(string(stringIn[0:]), char) {
		i += len(char)
		for i < len(stringIn) && !strings.HasPrefix(string(stringIn[i:]), char) {
			if strings.HasPrefix(string(stringIn[i:]), "\\") && i < len(stringIn)-1 {
				printChar(string(stringIn[i+1]), printIn)
				i++
				continue
			}
			printChar(string(stringIn[i]), printIn)
			i++
		}
		if strings.HasPrefix(string(stringIn[i:]), char) {
			i += len(char)
		}
		return i
	}
	return 0
}

func printChar(char string, printIn printConfig) {
	switch {
	case printIn.bold && printIn.italic:
		fmt.Print("\x1b[3;1m" + char)
	case printIn.italic:
		fmt.Print("\x1b[0;3m" + char)
	case printIn.bold:
		fmt.Print("\x1b[0;1m" + char)
	case printIn.code:
		fmt.Print("\x1b[2;37;100m" + char)
	default:
		fmt.Print("\x1b[0m" + char)
	}
}

func flatten(headerList []header, pref string) []header {
	returnList := []header{}
	for _, i := range headerList {
		nPref := i.header
		if pref != "" {
			nPref = pref + " - " + i.header
		}
		i.header = nPref
		returnList = append(returnList, i)
		if len(i.subHeaders) > 0 {
			returnList = append(returnList, flatten(i.subHeaders, nPref)...)
		}
	}
	return returnList
}
