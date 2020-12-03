package main

import (
	"fmt"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"net/http"
	"os"
	"os/exec"
)

var (
	resource = pflag.String("aad-resourcename", "https://management.azure.com/", "name of resource to grant token")
)

func greet(w http.ResponseWriter, r *http.Request) {
	log.Infof("Received request")
	hostname := os.Getenv("HOSTNAME")
	log.Infof("running request in sample pod %s", hostname)
	fmt.Fprintf(w, "Received request from: %q\n App name: %s\n Env variable: %s\n Name of the Pod: %s\n", r.RemoteAddr, os.Getenv("APPNAME"), os.Getenv("SomeVariable"), os.Getenv("HOSTNAME"))
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func main() {
    log.Infof("Starting up")
	cpCmd := exec.Command("cp", "-rf", "/env/some.env", ".")
	err := cpCmd.Run()
	if err != nil {
		log.Infof("Error loading .env path %s", "/env/some.env")
	}

	err = godotenv.Load("some.env")
	if err != nil {
		log.Infof("Error loading .env file %s", "some.env")
	}

	hostname := os.Getenv("HOSTNAME")

	logger := log.WithFields(log.Fields{
		"hostname": hostname,
	})

	log.Infof("running msi test")
	msiTest(logger)
	log.Infof("completed msi test")

	http.HandleFunc("/", greet)
	log.Infof("Listening for requests")
	http.ListenAndServe(getEnv("PORT", ":8080"), nil)
}

func msiTest(logger *log.Entry) {
	log.Infof("Parsing flags")
	pflag.Parse()
	log.Infof("Getting endpoint")
	msiEndpoint, err := adal.GetMSIVMEndpoint()
	if err != nil {
		logger.Fatalf("failed to get msiendpoint, %+v", err)
	}
	testMSIEndpoint(logger, msiEndpoint, *resource)
}

func testMSIEndpoint(logger *log.Entry, msiEndpoint, resource string) *adal.Token {
	log.Infof("getting sp token")
	spt, err := adal.NewServicePrincipalTokenFromMSI(msiEndpoint, resource)
	if err != nil {
		log.Infof("failed to get sp token")
		logger.Errorf("failed to acquire a token using the MSI VM extension, Error: %+v", err)
		return nil
	}
	if err := spt.Refresh(); err != nil {
		logger.Errorf("failed to refresh ServicePrincipalTokenFromMSI using the MSI VM extension, msiEndpoint(%s)", msiEndpoint)
		return nil
	}
	token := spt.Token()
	if token.IsZero() {
		logger.Errorf("zero token found, MSI VM extension, msiEndpoint(%s)", msiEndpoint)
		return nil
	}
	logger.Infof("successfully acquired a token using the MSI, msiEndpoint(%s)", msiEndpoint)
	log.Infof("returning token")
	return &token
}
