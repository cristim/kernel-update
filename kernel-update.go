//usr/bin/go run $0 $@ ; exit
package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"golang.org/x/net/html"
)

const kernelURLBase = "http://kernel.ubuntu.com/~kernel-ppa/mainline/"

var logger *log.Logger

type fileURLs struct {
	allHeaders         string
	currentArchHeaders string
	currentArchImage   string
}

func main() {
	logger = log.New(os.Stdout, "", log.Lshortfile)

	var imageFlavor string
	flag.StringVar(&imageFlavor, "flavor", "generic", "Kernel flavor: lowlatency|generic.")
	flag.Parse()

	htmlBodyReader := httpGet(kernelURLBase)

	defer htmlBodyReader.Close()

	latestVersion := parseLatestKernelVersion(htmlBodyReader)
	logger.Println(latestVersion)

	dir, err := ioutil.TempDir("", "")

	if err != nil {
		logger.Fatalln(err.Error)
	}

	logger.Println("Using temporary directory", dir)
	os.Chdir(dir)
	downloadPackages(kernelURLBase, latestVersion, imageFlavor)

	if err := installPackages(dir); err != nil {
		logger.Println(err.Error())
	}

	fmt.Println("Cleaning up...")
	err = os.RemoveAll(dir)

	if err != nil {
		fmt.Println(err.Error())
	}

}

func installPackages(dir string) error {

	globPath := filepath.Join(dir, "*.deb")
	_, err := filepath.Glob(globPath)

	if err != nil {
		logger.Println(err.Error())
		return err
	}

	sudoCmd := fmt.Sprintf("sudo dpkg -i %s", globPath)
	fmt.Printf("Executing '%s'\n", sudoCmd)
	cmd := exec.Command("/bin/sh", "-c", sudoCmd)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout

	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	return err
}

func httpGet(kernelURL string) io.ReadCloser {
	resp, err := http.Get(kernelURL)
	if err != nil {
		log.Fatalln("Failed to perform request")
	}
	return resp.Body
}

func parseLatestKernelVersion(reader io.Reader) string {
	var latestVersion string
	doc, err := html.Parse(reader)
	if err != nil {
		log.Fatal(err)
	}

	walkKernelVersionsTree(doc, &latestVersion)
	return latestVersion
}

// always sets the latestVersion global variable to the last a href
func walkKernelVersionsTree(n *html.Node, latestVersion *string) {

	if n.Type == html.ElementNode && n.Data == "a" {
		*latestVersion = n.Attr[0].Val
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walkKernelVersionsTree(c, latestVersion)
	}

}

func downloadPackages(kernelURLBase, latestVersion, flavor string) {
	files := getPackageFiles(kernelURLBase, latestVersion, flavor)
	downloadFromURL(kernelURLBase + latestVersion + files.allHeaders)
	downloadFromURL(kernelURLBase + latestVersion + files.currentArchHeaders)
	downloadFromURL(kernelURLBase + latestVersion + files.currentArchImage)
}

func downloadFromURL(url string) {
	tokens := strings.Split(url, "/")
	fileName := tokens[len(tokens)-1]
	fmt.Println("Downloading", url, "to", fileName)

	// TODO: check file existence first with io.IsExist
	output, err := os.Create(fileName)
	if err != nil {
		fmt.Println("Error while creating", fileName, "-", err)
		return
	}
	defer output.Close()

	response, err := http.Get(url)
	if err != nil {
		fmt.Println("Error while downloading", url, "-", err)
		return
	}
	defer response.Body.Close()

	n, err := io.Copy(output, response.Body)
	if err != nil {
		fmt.Println("Error while downloading", url, "-", err)
		return
	}
	fmt.Println(n, "bytes downloaded.")
}

func getPackageFiles(kernelURLBase, latestVersion, flavor string) fileURLs {
	var fileList fileURLs
	htmlBodyReader := httpGet(kernelURLBase + latestVersion)

	doc, err := html.Parse(htmlBodyReader)
	if err != nil {
		log.Fatal(err)
	}

	walkBuildsTree(doc, &fileList, flavor)
	return fileList
}

// always sets the latestVersion global variable to the last a href
func walkBuildsTree(n *html.Node, urls *fileURLs, flavor string) {

	if n.Type == html.ElementNode && n.Data == "a" {
		file := n.Attr[0].Val

		if strings.Contains(file, "headers") &&
			strings.Contains(file, "all") {
			urls.allHeaders = file
		}
		if strings.Contains(file, runtime.GOARCH) &&
			strings.Contains(file, flavor) {
			if strings.Contains(file, "headers") {
				urls.currentArchHeaders = file
			}
			if strings.Contains(file, "image") {
				urls.currentArchImage = file
			}
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walkBuildsTree(c, urls, flavor)
	}
}
