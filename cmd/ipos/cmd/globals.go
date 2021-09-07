package cmd

import (
	"crypto/x509"
	"os"
	"time"

	humanize "github.com/dustin/go-humanize"

	"github.com/storeros/ipos/cmd/ipos/crypto"
	xhttp "github.com/storeros/ipos/cmd/ipos/http"
	"github.com/storeros/ipos/pkg/auth"
	objectlock "github.com/storeros/ipos/pkg/bucket/object/lock"
	"github.com/storeros/ipos/pkg/pubsub"
)

const (
	globalIPOSCertExpireWarnDays = time.Hour * 24 * 30

	globalIPOSDefaultPort = "9000"

	globalIPOSDefaultRegion       = ""
	globalIPOSDefaultOwnerID      = "02d6176db174dc93cb1b899f7c6078f08654445fe8cf1b6ce98d8855f66bdbf4"
	globalIPOSDefaultStorageClass = "STANDARD"
	globalWindowsOSName           = "windows"
)

const (
	maxFormFieldSize = int64(1 * humanize.MiByte)

	globalMaxSkewTime = 15 * time.Minute

	globalRefreshIAMInterval = 5 * time.Minute

	maxLocationConstraintSize = 3 * humanize.MiByte
)

var globalCLIContext = struct {
	JSON, Quiet    bool
	Anonymous      bool
	Addr           string
	StrictS3Compat bool
}{}

var (
	globalBrowserEnabled = true

	globalServerRegion = globalIPOSDefaultRegion

	globalIPOSAddr = ""
	globalIPOSPort = globalIPOSDefaultPort
	globalIPOSHost = ""

	globalPolicySys *PolicySys
	globalIAMSys    *IAMSys

	globalAPIThrottling apiThrottling

	globalRootCAs *x509.CertPool

	globalIsSSL bool

	globalHTTPServer        *xhttp.Server
	globalHTTPServerErrorCh = make(chan error)
	globalOSSignalCh        = make(chan os.Signal, 1)

	globalHTTPTrace = pubsub.New()

	globalConsoleSys *HTTPConsoleLoggerSys

	globalEndpoints Endpoints

	globalHTTPStats = newHTTPStats()

	globalActiveCred auth.Credentials

	globalOldCred auth.Credentials

	globalConfigEncrypted bool

	globalDomainNames []string

	globalBucketObjectLockConfig = objectlock.NewBucketObjectLockConfig()

	GlobalKMS crypto.KMS

	globalAutoEncryption bool

	standardExcludeCompressExtensions = []string{".gz", ".bz2", ".rar", ".zip", ".7z", ".xz", ".mp4", ".mkv", ".mov"}

	standardExcludeCompressContentTypes = []string{"video/*", "audio/*", "application/zip", "application/x-gzip", "application/x-zip-compressed", " application/x-compress", "application/x-spoon"}

	globalDeploymentID string
)

func getGlobalInfo() (globalInfo map[string]interface{}) {
	globalInfo = map[string]interface{}{
		"serverRegion": globalServerRegion,
	}

	return globalInfo
}
