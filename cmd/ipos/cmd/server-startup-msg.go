package cmd

import (
	"crypto/x509"
	"fmt"
	"net"
	"runtime"
	"strings"

	color "github.com/storeros/ipos/pkg/color"
	xnet "github.com/storeros/ipos/pkg/net"
)

func getFormatStr(strLen int, padding int) string {
	formatStr := fmt.Sprintf("%ds", strLen+padding)
	return "%" + formatStr
}

func printStartupSafeModeMessage(apiEndpoints []string, err error) {
	logStartupMessage(color.RedBold("Server startup failed with '%v'", err))
	logStartupMessage(color.RedBold("Server switching to safe mode"))
	logStartupMessage(color.RedBold("Please use 'mc admin config' commands fix this issue"))

	cred := globalActiveCred

	region := globalServerRegion

	strippedAPIEndpoints := stripStandardPorts(apiEndpoints)

	apiEndpointStr := strings.Join(strippedAPIEndpoints, "  ")

	logStartupMessage(color.Red("Endpoint: ") + color.Bold(fmt.Sprintf(getFormatStr(len(apiEndpointStr), 1), apiEndpointStr)))
	if color.IsTerminal() && !globalCLIContext.Anonymous {
		logStartupMessage(color.Red("AccessKey: ") + color.Bold(fmt.Sprintf("%s ", cred.AccessKey)))
		logStartupMessage(color.Red("SecretKey: ") + color.Bold(fmt.Sprintf("%s ", cred.SecretKey)))
		if region != "" {
			logStartupMessage(color.Red("Region: ") + color.Bold(fmt.Sprintf(getFormatStr(len(region), 3), region)))
		}
	}

	alias := "myipos"
	endPoint := strippedAPIEndpoints[0]

	if color.IsTerminal() && !globalCLIContext.Anonymous {
		logStartupMessage(color.RedBold("\nCommand-line Access: "))
		if runtime.GOOS == globalWindowsOSName {
			mcMessage := fmt.Sprintf("> mc.exe config host add %s %s %s %s --api s3v4", alias,
				endPoint, cred.AccessKey, cred.SecretKey)
			logStartupMessage(fmt.Sprintf(getFormatStr(len(mcMessage), 3), mcMessage))
			mcMessage = "> mc.exe admin config --help"
			logStartupMessage(fmt.Sprintf(getFormatStr(len(mcMessage), 3), mcMessage))
		} else {
			mcMessage := fmt.Sprintf("$ mc config host add %s %s %s %s --api s3v4", alias,
				endPoint, cred.AccessKey, cred.SecretKey)
			logStartupMessage(fmt.Sprintf(getFormatStr(len(mcMessage), 3), mcMessage))
			mcMessage = "$ mc admin config --help"
			logStartupMessage(fmt.Sprintf(getFormatStr(len(mcMessage), 3), mcMessage))
		}
	}
}

func printStartupMessage(apiEndpoints []string) {
	strippedAPIEndpoints := stripStandardPorts(apiEndpoints)
	printServerCommonMsg(strippedAPIEndpoints)
}

func isNotIPv4(host string) bool {
	h, _, err := net.SplitHostPort(host)
	if err != nil {
		h = host
	}
	ip := net.ParseIP(h)
	ok := ip.To4() != nil

	return !ok
}

func stripStandardPorts(apiEndpoints []string) (newAPIEndpoints []string) {
	newAPIEndpoints = make([]string, len(apiEndpoints))
	for i, apiEndpoint := range apiEndpoints {
		u, err := xnet.ParseHTTPURL(apiEndpoint)
		if err != nil {
			continue
		}
		if globalIPOSHost == "" && isNotIPv4(u.Host) {
			continue
		}
		newAPIEndpoints[i] = u.String()
	}
	return newAPIEndpoints
}

func printServerCommonMsg(apiEndpoints []string) {
	cred := globalActiveCred

	region := globalServerRegion

	apiEndpointStr := strings.Join(apiEndpoints, "  ")

	logStartupMessage(color.Blue("Endpoint: ") + color.Bold(fmt.Sprintf(getFormatStr(len(apiEndpointStr), 1), apiEndpointStr)))
	if color.IsTerminal() && !globalCLIContext.Anonymous {
		logStartupMessage(color.Blue("AccessKey: ") + color.Bold(fmt.Sprintf("%s ", cred.AccessKey)))
		logStartupMessage(color.Blue("SecretKey: ") + color.Bold(fmt.Sprintf("%s ", cred.SecretKey)))
		if region != "" {
			logStartupMessage(color.Blue("Region: ") + color.Bold(fmt.Sprintf(getFormatStr(len(region), 3), region)))
		}
	}

	if globalBrowserEnabled {
		logStartupMessage(color.Blue("\nBrowser Access:"))
		logStartupMessage(fmt.Sprintf(getFormatStr(len(apiEndpointStr), 3), apiEndpointStr))
	}
}

func getCertificateChainMsg(certs []*x509.Certificate) string {
	msg := color.Blue("\nCertificate expiry info:\n")
	totalCerts := len(certs)
	var expiringCerts int
	for i := totalCerts - 1; i >= 0; i-- {
		cert := certs[i]
		if cert.NotAfter.Before(UTCNow().Add(globalIPOSCertExpireWarnDays)) {
			expiringCerts++
			msg += fmt.Sprintf(color.Bold("#%d %s will expire on %s\n"), expiringCerts, cert.Subject.CommonName, cert.NotAfter)
		}
	}
	if expiringCerts > 0 {
		return msg
	}
	return ""
}

func printCertificateMsg(certs []*x509.Certificate) {
	logStartupMessage(getCertificateChainMsg(certs))
}
