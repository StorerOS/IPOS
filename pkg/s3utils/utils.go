package s3utils

import (
	"bytes"
	"encoding/hex"
	"errors"
	"net"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

var sentinelURL = url.URL{}

func IsValidDomain(host string) bool {
	host = strings.TrimSpace(host)
	if len(host) == 0 || len(host) > 255 {
		return false
	}

	if host[len(host)-1:] == "-" || host[:1] == "-" {
		return false
	}

	if host[len(host)-1:] == "_" || host[:1] == "_" {
		return false
	}

	if host[:1] == "." {
		return false
	}

	if strings.ContainsAny(host, "`~!@#$%^&*()+={}[]|\\\"';:><?/") {
		return false
	}

	return true
}

func IsValidIP(ip string) bool {
	return net.ParseIP(ip) != nil
}

func IsVirtualHostSupported(endpointURL url.URL, bucketName string) bool {
	if endpointURL == sentinelURL {
		return false
	}

	if endpointURL.Scheme == "https" && strings.Contains(bucketName, ".") {
		return false
	}

	return IsAmazonEndpoint(endpointURL) || IsGoogleEndpoint(endpointURL) || IsAliyunOSSEndpoint(endpointURL)
}

var amazonS3HostHyphen = regexp.MustCompile(`^s3-(.*?).amazonaws.com$`)

var amazonS3HostDualStack = regexp.MustCompile(`^s3.dualstack.(.*?).amazonaws.com$`)

var amazonS3HostDot = regexp.MustCompile(`^s3.(.*?).amazonaws.com$`)

var amazonS3ChinaHost = regexp.MustCompile(`^s3.(cn.*?).amazonaws.com.cn$`)

var elbAmazonRegex = regexp.MustCompile(`elb(.*?).amazonaws.com$`)

var elbAmazonCnRegex = regexp.MustCompile(`elb(.*?).amazonaws.com.cn$`)

func GetRegionFromURL(endpointURL url.URL) string {
	if endpointURL == sentinelURL {
		return ""
	}
	if endpointURL.Host == "s3-external-1.amazonaws.com" {
		return ""
	}
	if IsAmazonGovCloudEndpoint(endpointURL) {
		return "us-gov-west-1"
	}

	if elbAmazonRegex.MatchString(endpointURL.Host) || elbAmazonCnRegex.MatchString(endpointURL.Host) {
		return ""
	}
	parts := amazonS3HostDualStack.FindStringSubmatch(endpointURL.Host)
	if len(parts) > 1 {
		return parts[1]
	}
	parts = amazonS3HostHyphen.FindStringSubmatch(endpointURL.Host)
	if len(parts) > 1 {
		return parts[1]
	}
	parts = amazonS3ChinaHost.FindStringSubmatch(endpointURL.Host)
	if len(parts) > 1 {
		return parts[1]
	}
	parts = amazonS3HostDot.FindStringSubmatch(endpointURL.Host)
	if len(parts) > 1 {
		return parts[1]
	}
	return ""
}

func IsAliyunOSSEndpoint(endpointURL url.URL) bool {
	return strings.HasSuffix(endpointURL.Host, "aliyuncs.com")
}

func IsAmazonEndpoint(endpointURL url.URL) bool {
	if endpointURL.Host == "s3-external-1.amazonaws.com" || endpointURL.Host == "s3.amazonaws.com" {
		return true
	}
	return GetRegionFromURL(endpointURL) != ""
}

func IsAmazonGovCloudEndpoint(endpointURL url.URL) bool {
	if endpointURL == sentinelURL {
		return false
	}
	return (endpointURL.Host == "s3-us-gov-west-1.amazonaws.com" ||
		IsAmazonFIPSGovCloudEndpoint(endpointURL))
}

func IsAmazonFIPSGovCloudEndpoint(endpointURL url.URL) bool {
	if endpointURL == sentinelURL {
		return false
	}
	return endpointURL.Host == "s3-fips-us-gov-west-1.amazonaws.com" ||
		endpointURL.Host == "s3-fips.dualstack.us-gov-west-1.amazonaws.com"
}

func IsAmazonFIPSUSEastWestEndpoint(endpointURL url.URL) bool {
	if endpointURL == sentinelURL {
		return false
	}
	switch endpointURL.Host {
	case "s3-fips.us-east-2.amazonaws.com":
	case "s3-fips.dualstack.us-west-1.amazonaws.com":
	case "s3-fips.dualstack.us-west-2.amazonaws.com":
	case "s3-fips.dualstack.us-east-2.amazonaws.com":
	case "s3-fips.dualstack.us-east-1.amazonaws.com":
	case "s3-fips.us-west-1.amazonaws.com":
	case "s3-fips.us-west-2.amazonaws.com":
	case "s3-fips.us-east-1.amazonaws.com":
	default:
		return false
	}
	return true
}

func IsAmazonFIPSEndpoint(endpointURL url.URL) bool {
	return IsAmazonFIPSUSEastWestEndpoint(endpointURL) || IsAmazonFIPSGovCloudEndpoint(endpointURL)
}

func IsGoogleEndpoint(endpointURL url.URL) bool {
	if endpointURL == sentinelURL {
		return false
	}
	return endpointURL.Host == "storage.googleapis.com"
}

func percentEncodeSlash(s string) string {
	return strings.Replace(s, "/", "%2F", -1)
}

func QueryEncode(v url.Values) string {
	if v == nil {
		return ""
	}
	var buf bytes.Buffer
	keys := make([]string, 0, len(v))
	for k := range v {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		vs := v[k]
		prefix := percentEncodeSlash(EncodePath(k)) + "="
		for _, v := range vs {
			if buf.Len() > 0 {
				buf.WriteByte('&')
			}
			buf.WriteString(prefix)
			buf.WriteString(percentEncodeSlash(EncodePath(v)))
		}
	}
	return buf.String()
}

func TagDecode(ctag string) map[string]string {
	if ctag == "" {
		return map[string]string{}
	}
	tags := strings.Split(ctag, "&")
	tagMap := make(map[string]string, len(tags))
	var err error
	for _, tag := range tags {
		kvs := strings.SplitN(tag, "=", 2)
		if len(kvs) == 0 {
			return map[string]string{}
		}
		if len(kvs) == 1 {
			return map[string]string{}
		}
		tagMap[kvs[0]], err = url.PathUnescape(kvs[1])
		if err != nil {
			continue
		}
	}
	return tagMap
}

func TagEncode(tags map[string]string) string {
	values := url.Values{}
	for k, v := range tags {
		values[k] = []string{v}
	}
	return QueryEncode(values)
}

var reservedObjectNames = regexp.MustCompile("^[a-zA-Z0-9-_.~/]+$")

func EncodePath(pathName string) string {
	if reservedObjectNames.MatchString(pathName) {
		return pathName
	}
	var encodedPathname string
	for _, s := range pathName {
		if 'A' <= s && s <= 'Z' || 'a' <= s && s <= 'z' || '0' <= s && s <= '9' {
			encodedPathname = encodedPathname + string(s)
			continue
		}
		switch s {
		case '-', '_', '.', '~', '/':
			encodedPathname = encodedPathname + string(s)
			continue
		default:
			len := utf8.RuneLen(s)
			if len < 0 {
				return pathName
			}
			u := make([]byte, len)
			utf8.EncodeRune(u, s)
			for _, r := range u {
				hex := hex.EncodeToString([]byte{r})
				encodedPathname = encodedPathname + "%" + strings.ToUpper(hex)
			}
		}
	}
	return encodedPathname
}

var (
	validBucketName       = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9\.\-\_\:]{1,61}[A-Za-z0-9]$`)
	validBucketNameStrict = regexp.MustCompile(`^[a-z0-9][a-z0-9\.\-]{1,61}[a-z0-9]$`)
	ipAddress             = regexp.MustCompile(`^(\d+\.){3}\d+$`)
)

func checkBucketNameCommon(bucketName string, strict bool) (err error) {
	if strings.TrimSpace(bucketName) == "" {
		return errors.New("Bucket name cannot be empty")
	}
	if len(bucketName) < 3 {
		return errors.New("Bucket name cannot be shorter than 3 characters")
	}
	if len(bucketName) > 63 {
		return errors.New("Bucket name cannot be longer than 63 characters")
	}
	if ipAddress.MatchString(bucketName) {
		return errors.New("Bucket name cannot be an ip address")
	}
	if strings.Contains(bucketName, "..") || strings.Contains(bucketName, ".-") || strings.Contains(bucketName, "-.") {
		return errors.New("Bucket name contains invalid characters")
	}
	if strict {
		if !validBucketNameStrict.MatchString(bucketName) {
			err = errors.New("Bucket name contains invalid characters")
		}
		return err
	}
	if !validBucketName.MatchString(bucketName) {
		err = errors.New("Bucket name contains invalid characters")
	}
	return err
}

func CheckValidBucketName(bucketName string) (err error) {
	return checkBucketNameCommon(bucketName, false)
}

func CheckValidBucketNameStrict(bucketName string) (err error) {
	return checkBucketNameCommon(bucketName, true)
}

func CheckValidObjectNamePrefix(objectName string) error {
	if len(objectName) > 1024 {
		return errors.New("Object name cannot be longer than 1024 characters")
	}
	if !utf8.ValidString(objectName) {
		return errors.New("Object name with non UTF-8 strings are not supported")
	}
	return nil
}

func CheckValidObjectName(objectName string) error {
	if strings.TrimSpace(objectName) == "" {
		return errors.New("Object name cannot be empty")
	}
	return CheckValidObjectNamePrefix(objectName)
}
