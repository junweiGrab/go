package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/tebeka/selenium"
	"github.com/tebeka/selenium/chrome"
)

var (
	help = `Usage: ./login_gmail [fileDir] [chromeDriver]
        - fileDir: CSV file dir which contains the gmail
        - chromeDriver: executable of chromeDriver
                - chromeBinary: binary of chrome`
)

// Status of login result
const (
	PWDCHANGED = iota
	PWDNOTCHANGED
	NEEDVERIFICATION
	INVALID
	FAILED
)

func main() {

	args := os.Args[1:]
	if len(args) != 3 {
		fmt.Println(help)
		return
	}
	fileDir := args[0]
	fmt.Printf("file dir: %s\n", fileDir)
	files, err := ioutil.ReadDir(fileDir)
	if err != nil {
		panic(err)
	}
	fmt.Printf("#%d of files\n", len(files))
	wdPath := args[1]
	browserPath := args[2]
	fmt.Printf("wd path: %s, browser path: %s\n", wdPath, browserPath)

	port := 9666
	var wg sync.WaitGroup
	wg.Add(len(files))
	for index, f := range files {
		if f.IsDir() {
			continue
		}
		go func(port int, fileName string) {
			defer wg.Done()
			do(fileName, wdPath, browserPath, port)
		}(index+port, fileDir+"/"+f.Name())
	}
	wg.Wait()
}
func start(executable string, port int, killed <-chan bool, done chan<- bool) {
	command := exec.Command(executable, fmt.Sprintf("--port=%d", port))
	fmt.Printf("[webdriver] starting with command: %v\n", command)
	err := command.Start()
	if err != nil {
		fmt.Printf("[webdriver] start up webdriver error: %v\n", err)
		panic(err)
	}
	time.Sleep(time.Second * 5)
	fmt.Printf("[webdriver] started.\n")
	done <- true
	<-killed
	fmt.Printf("[webdirver] shutted down.\n")
}

func do(fileName, wdPath, browswerPath string, port int) {
	killed := make(chan bool)
	done := make(chan bool)
	// start a webdriver.
	go start(wdPath, port, killed, done)
	<-done
	emails, err := readEmail(fileName)
	if err != nil {
		killed <- true
		fmt.Printf("Error when read email from file[%s]: %v\n", fileName, err)
		return
	}
	// Connect to the WebDriver instance running locally.
	caps := selenium.Capabilities{
		"browserName": "chrome",
	}
	chromeCapOptions := chrome.Capabilities{
		Path: browswerPath,
		Args: []string{"--headless", "--disable-gpu"},
	}
	caps.AddChrome(chromeCapOptions)

	outputDir := filepath.Dir(fileName) + "/output"
	os.Mkdir(outputDir, os.ModePerm)
	name := filepath.Base(fileName)
	pwdChanged, _ := os.Create(fmt.Sprintf("%s/%s-pwd-changed", outputDir, name))
	defer pwdChanged.Close()
	pwdNotChanged, _ := os.Create(fmt.Sprintf("%s/%s-pwd-not-changed", outputDir, name))
	defer pwdNotChanged.Close()
	needVerify, _ := os.Create(fmt.Sprintf("%s/%s-need-verify", outputDir, name))
	defer needVerify.Close()
	invalid, _ := os.Create(fmt.Sprintf("%s/%s-invalid", outputDir, name))
	defer invalid.Close()
	failed, _ := os.Create(fmt.Sprintf("%s/%s-failed", outputDir, name))
	defer failed.Close()

	for _, email := range emails {
		statusID := login(email, "popolopo", caps, port)
		switch statusID {
		case PWDCHANGED:
			pwdChanged.WriteString(email + "\n")
		case PWDNOTCHANGED:
			pwdNotChanged.WriteString(email + "\n")
		case NEEDVERIFICATION:
			needVerify.WriteString(email + "\n")
		case INVALID:
			invalid.WriteString(email + "\n")
		case FAILED:
			failed.WriteString(email + "\n")
		}
	}
	// classification(fileName, pwdChangedEmails, pwdNotChangedEmails, needManualVerifyEmails, invalidEmails, failedEmails) // kill the webdriver from running.
	killed <- true
}

func readEmail(fileName string) ([]string, error) {
	fmt.Printf("Reading email from file: %s\n", fileName)
	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	emails := []string{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		emails = append(emails, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return emails, nil
}

func login(email, password string, caps selenium.Capabilities, port int) int {
	wd, err := selenium.NewRemote(caps, fmt.Sprintf("http://localhost:%d", port))
	if err != nil {
		fmt.Printf("FAILED when new a remote web driver [%s]: %s\n", email, err)
		return FAILED
	}
	defer wd.Quit()

	// Navigate to the simple playground interface.
	if err := wd.Get("http://www.gmail.com"); err != nil {
		fmt.Printf("FAILED when open gmail.com [%s]: %s\n", email, err)
		return FAILED
	}
	// Enter userd id
	elem, err := wd.FindElement(selenium.ByXPATH, `//*[@id="identifierId"]`)
	if err != nil {
		fmt.Printf("FAILED when find identifier field [%s]: %s\n", email, err)
		return FAILED
	}
	err = elem.SendKeys(email)
	if err != nil {
		fmt.Printf("FAILED when input email address [%s]: %s\n", email, err)
		return FAILED
	}

	btn, err := elem.FindElement(selenium.ByXPATH, `//*[@id="identifierNext"]`)
	if err != nil {
		fmt.Printf("FAILED when find identifier next button [%s]: %s\n", email, err)
		return FAILED
	}
	if err := btn.Click(); err != nil {
		fmt.Printf("FAILED when click identifier next button [%s]: %s\n", email, err)
		return FAILED
	}

	time.Sleep(time.Second * 2)
	// Enter userd id
	elem, err = wd.FindElement(selenium.ByXPATH, `//*[@id="password"]/div[1]/div/div[1]/input`)
	if err != nil {
		fmt.Printf("INVALID when find password field [%s]: %s\n", email, err)
		return INVALID
	}
	err = elem.SendKeys(password)
	if err != nil {
		fmt.Printf("INVALID when input password [%s]: %s\n", email, err)
		return INVALID
	}

	btn, err = elem.FindElement(selenium.ByXPATH, `//*[@id="passwordNext"]`)
	if err != nil {
		fmt.Printf("INVALID when find password next button [%s]: %s\n", email, err)
		return INVALID
	}
	// record current url before click button.
	if err := btn.Click(); err != nil {
		fmt.Printf("FAILED when click password next button [%s]: %s\n", email, err)
		return FAILED
	}
	// // get current url after click
	time.Sleep(time.Millisecond * 500)

	ca, err := wd.FindElement(selenium.ByXPATH, `//*[@id="captchaimg"]`)
	if err == nil && ca != nil {
		if src, _ := ca.GetAttribute("src"); src != "" {
			fmt.Printf("NV [%s], img src: %s\n", email, src)
			return NEEDVERIFICATION
		}
	}
	msgElem, err := wd.FindElement(selenium.ByCSSSelector, `#password > div.LXRPh > div.dEOOab.RxsGPe`)
	// the request is blocked in current page, and show some message.
	if err == nil && msgElem != nil {
		txt, _ := msgElem.Text()
		if txt != "" {
			fmt.Printf("CHANGED [%s]: %s\n", email, txt)
			return PWDCHANGED
		}
	}
	// time.Sleep(time.Second * 2)
	// } else if afterLoginURL == "https://mail.google.com/mail/u/0/#inbox" {
	//      return PWDNOTCHANGED

	fmt.Printf("NOT-CHANGED [%s]\n", email)
	return PWDNOTCHANGED
}
