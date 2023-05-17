package lib

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
    "encoding/pem"
    "crypto/tls"
)


type HttpClientConf struct {
	httpClient *http.Client
}

func GetHeader(url string) (http.Client, *http.Request) {

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println("Error with http.NewRequest creation")
	}
	req.Header.Set("accept", "text/xml")

	return getHttpClient(), req

}

// getHttpClient() is a singleton and should always give same reference
// request made with httpClient will get timed out after 60sec
var httpClientConfInstance HttpClientConf
func getHttpClient() http.Client {
	if httpClientConfInstance.httpClient == nil {
		httpClientConfInstance = *new(HttpClientConf)
		conf := getTLS()
		httpClientConfInstance.httpClient = &http.Client{ Timeout: 60 * time.Second, Transport: &http.Transport{TLSClientConfig: conf} }
	}
	return *(httpClientConfInstance.httpClient)
}



type TLSConf struct {
	tlsConfig *tls.Config
}

// getTLS() is a singleton and should always give same reference
var tlsConfigGlobalInstance TLSConf
func getTLS() *tls.Config {
	if tlsConfigGlobalInstance.tlsConfig == nil {
		tlsConfigGlobalInstance = *new(TLSConf)
		tlsConfigGlobalInstance.tlsConfig = setupTLS()
	}
	return tlsConfigGlobalInstance.tlsConfig
}

func setupTLS() *tls.Config {
	// use ssl key/create_cert.go to create certificate and key or
	// openssl req -new -x509 -key privateKey.pem -out certificate.crt -days 365

	var PEM_KEY = "-----BEGIN EC PRIVATE KEY-----\n" +
	"MHcCAQEEIEFRa42BSz1uuRxWBh60vePDrpkgtELJJMZtkJGlExuLoAoGCCqGSM49\n" +
	"AwEHoUQDQgAEyiUJYA7SI/u2Rf8ouND0Ip46gdjKcGB8Vx3VkajFx5+YhtaMfHb1\n" +
	"5YjfGWFuNLqyxLGGvDUq6HlGsBJ9QIcPtA==\n" +
	"-----END EC PRIVATE KEY-----\n"

	var PEM_CERT = "-----BEGIN CERTIFICATE-----\n" +
	"MIIBHjCBxaADAgECAgEBMAoGCCqGSM49BAMCMBcxFTATBgNVBAoTDERvY2tlciwg\n" +
	"SW5jLjAeFw0xMzA3MjUwMTEwMjRaFw0xNTA3MjUwMTEwMjRaMBcxFTATBgNVBAoT\n" +
	"DERvY2tlciwgSW5jLjBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABMolCWAO0iP7\n" +
	"tkX/KLjQ9CKeOoHYynBgfFcd1ZGoxcefmIbWjHx29eWI3xlhbjS6ssSxhrw1Kuh5\n" +
	"RrASfUCHD7SjAjAAMAoGCCqGSM49BAMCA0gAMEUCIQDRLQTSSeqjsxsb+q4exLSt\n" +
	"EM7f7/ymBzoUzbXU7wI9AgIgXCWaI++GkopGT8T2qV/3+NL0U+fYM0ZjSNSiwaK3\n" +
	"+kA=\n" +
	"-----END CERTIFICATE-----\n"


    // Try to load bundle with private and public key
    bundle, err := ioutil.ReadFile("ssl key\\certificate.crt")
    if err != nil {
        fmt.Println("Error with loading certificate.crt bundle from ssl key folder; Using default keys.\n")
    } else {

	    keyBlock, _ := pem.Decode(bundle)

	    if keyBlock == nil || keyBlock.Type != "CERTIFICATE" {
	        fmt.Println("keyBlock doesn't have CERTIFICATE Block")
	        panic("Error with TLS creation")
	    } else {
		    // Load bundle with private EC key
		    bundle2, err := ioutil.ReadFile("ssl key\\privateKey.pem")
		    if err != nil {
		        fmt.Println("Error with loading privateKey.pem bundle")
		        panic(err)
		    }

		    keyBlock2, _ := pem.Decode(bundle2)

		    if keyBlock2 == nil || keyBlock2.Type != "EC PRIVATE KEY" {
		        fmt.Println("privateKey.pem keyBlock doesn't have EC PRIVATE KEY Block")
		        panic("Error with TLS creation")
		    }

			PEM_CERT = string(bundle)
			PEM_KEY = string(bundle2)
		}

	}


    cert, err := tls.X509KeyPair([]byte(PEM_CERT), []byte(PEM_KEY))
    if err != nil {
        fmt.Println("Error with tls X509 key pair creation")
        panic(err)
    }

    config := &tls.Config{
        Certificates: []tls.Certificate{cert},
    }

    return config
}