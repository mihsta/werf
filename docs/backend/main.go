package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

type ChannelType struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type ReleaseType struct {
	Group    string        `json:"group"`
	Channels []ChannelType `json:"channels"`
}

type ReleasesStatusType struct {
	Releases []ReleaseType `json:"multiwerf"`
}

type ApiStatusResponseType struct {
	Status         string        `json:"status"`
	Msg            string        `json:"msg"`
	RootVersion    string        `json:"rootVersion"`
	RootVersionURL string        `json:"rootVersionURL"`
	Multiwerf      []ReleaseType `json:"multiwerf"`
}

type versionMenuType struct {
	VersionItems      []versionMenuItems
	HTMLContent       string
	CurrentGroup      string
	CurrentChannel    string
	CurrentVersion    string
	CurrentVersionURL string
	CurrentPageURL    string // Page URL menu requesting for with a leading /documentation/, e.g /documentation/reference/build_process.html. Or "/documentation/" for unknown cases.
}

type versionMenuItems struct {
	Group      string
	Channel    string
	Version    string
	VersionURL string // Base URL for corresponding version without a leading /, e.g. 'v1.2.3-plus-fix6'.
	IsCurrent  bool
}

var ReleasesStatus ReleasesStatusType

var channelsListReverseStability = []string{"rock-solid", "stable", "ea", "beta", "alpha"}

func getPage(filename string) ([]byte, error) {
	body, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return body, nil
}

// Retrieves page URL menu requested for with a leading /documentation/
// E.g /documentation/reference/build_process.html. Or "/documentation/" for unknown cases.
func getCurrentPageURL(r *http.Request) (result string) {
	result = "/documentation/"
	originalURI := r.Header.Get("x-original-uri")
	if strings.Contains(originalURI, "/documentation/") {
		items := strings.Split(originalURI, "/documentation/")
		if len(items) > 1 {
			result += strings.Join(items[1:], "/documentation/")
		}
	}
	return
}

// Retrieves version URL page belongs to if request came from concrete documentation version, otherwise empty.
// E.g for /v1.2.3-plus-fix5/documentation/reference/build_process.html return "v1.2.3-plus-fix5".
func getVersionURL(r *http.Request) (result string) {
	result = ""
	originalURI := r.Header.Get("x-original-uri")
	if strings.Contains(originalURI, "/documentation/") {
		result = strings.Split(originalURI, "/documentation/")[0]
	}
	return strings.TrimPrefix(result, "/")
}

func inspectRequest(r *http.Request) string {
	var request []string

	url := fmt.Sprintf("%v %v %v", r.Method, r.URL, r.Proto)
	request = append(request, url)
	request = append(request, fmt.Sprintf("Host: %v", r.Host))
	for name, headers := range r.Header {
		name = strings.ToLower(name)
		for _, h := range headers {
			request = append(request, fmt.Sprintf("%v: %v", name, h))
		}
	}

	// If this is a POST, add post data
	if r.Method == "POST" {
		_ = r.ParseForm()
		request = append(request, "\n")
		request = append(request, r.Form.Encode())
	}

	return strings.Join(request, "\n")
}

func inspectRequestHTML(r *http.Request) string {
	var request []string

	request = append(request, "<p>")
	url := fmt.Sprintf("<b>%v</b> %v %v", r.Method, r.URL, r.Proto)
	request = append(request, url)
	request = append(request, fmt.Sprintf("<b>Host:</b> %v", r.Host))
	for name, headers := range r.Header {
		name = strings.ToLower(name)
		for _, h := range headers {
			request = append(request, fmt.Sprintf("<b>%v:</b> %v", name, h))
		}
	}

	// If this is a POST, add post data
	if r.Method == "POST" {
		_ = r.ParseForm()
		request = append(request, r.Form.Encode())
	}

	request = append(request, "</p>")
	return strings.Join(request, "<br />")
}

func VersionToURL(version string) string {
	result := strings.ReplaceAll(version, "+", "-plus-")
	result = strings.ReplaceAll(result, "_", "-u-")
	return result
}

func URLToVersion(version string) (result string) {
	result = strings.ReplaceAll(version, "-plus-", "+")
	result = strings.ReplaceAll(result, "-u-", "_")
	return
}

func getRootReleaseVersion() string {
	var activeRelease string

	if len(os.Getenv("ACTIVE_RELEASE")) > 0 {
		activeRelease = os.Getenv("ACTIVE_RELEASE")
	} else {
		activeRelease = "1.2"
	}

	_ = updateReleasesStatus()

	if len(ReleasesStatus.Releases) > 0 {
		for _, ReleaseGroup := range ReleasesStatus.Releases {
			if ReleaseGroup.Group == activeRelease {
				releaseVersions := make(map[string]string)
				for _, channel := range ReleaseGroup.Channels {
					releaseVersions[channel.Name] = channel.Version
				}

				if _, ok := releaseVersions["stable"]; ok {
					return releaseVersions["stable"]
				} else if _, ok := releaseVersions["ea"]; ok {
					return releaseVersions["ea"]
				} else if _, ok := releaseVersions["beta"]; ok {
					return releaseVersions["beta"]
				} else if _, ok := releaseVersions["alpha"]; ok {
					return releaseVersions["alpha"]
				}
			}
		}
	}
	return "unknown"
}

func apiStatusHandler(w http.ResponseWriter, r *http.Request) {
	var msg []string
	status := "ok"

	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	err := updateReleasesStatus()
	if err != nil {
		msg = append(msg, err.Error())
		status = "error"
	}

	_ = json.NewEncoder(w).Encode(
		ApiStatusResponseType{
			Status:         status,
			Msg:            strings.Join(msg, " "),
			RootVersion:    getRootReleaseVersion(),
			RootVersionURL: VersionToURL(getRootReleaseVersion()),
			Multiwerf:      ReleasesStatus.Releases,
		})
}

func documentationHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Accel-Redirect", fmt.Sprintf("/%v%v", VersionToURL(getRootReleaseVersion()), r.URL.RequestURI()))
}

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func topnavHandler(w http.ResponseWriter, r *http.Request) {
	_ = updateReleasesStatus()

	versionMenu := versionMenuType{
		VersionItems:      []versionMenuItems{},
		HTMLContent:       "",
		CurrentGroup:      "",
		CurrentChannel:    "",
		CurrentVersion:    "",
		CurrentVersionURL: "",
		CurrentPageURL:    "",
	}

	_ = versionMenu.getVersionMenuData(r, &ReleasesStatus)

	tplPath := "./root/main/includes/topnav.html"
	if strings.HasPrefix(r.Host, "ru.") {
		tplPath = "./root/ru/includes/topnav.html"
	}
	tpl := template.Must(template.ParseFiles(tplPath))
	err := tpl.Execute(w, versionMenu)
	if err != nil {
		// Log error or maybe make some magic?
		http.Error(w, "Internal Server Error (template error)", 500)
	}
}

func (m *versionMenuType) getVersionMenuData(r *http.Request, releases *ReleasesStatusType) (err error) {
	err = nil

	m.CurrentPageURL = getCurrentPageURL(r)
	m.CurrentVersionURL = getVersionURL(r)
	m.CurrentVersion = URLToVersion(m.CurrentVersionURL)

	if m.CurrentVersion == "" {
		m.CurrentVersion = getRootReleaseVersion()
		m.CurrentVersionURL = VersionToURL(m.CurrentVersion)
	}

	// Try to find current channel from URL
	m.CurrentChannel, m.CurrentGroup = getChannelAndGroupFromVersion(&ReleasesStatus, m.CurrentVersion)

	// Add the first menu item
	m.VersionItems = append(m.VersionItems, versionMenuItems{
		Group:      m.CurrentGroup,
		Channel:    m.CurrentChannel,
		Version:    m.CurrentVersion,
		VersionURL: m.CurrentVersionURL,
		IsCurrent:  true,
	})

	//for _, releaseItem_ := range releases.Releases {
	//	if releaseItem_.Group == m.CurrentGroup {
	//		for _, channelItem_ := range releaseItem_.Channels {
	//			if channelItem_.Name == m.CurrentChannel {
	//				m.VersionItems = append(m.VersionItems, versionMenuItems{
	//					Group:      m.CurrentGroup,
	//					Channel:    m.CurrentChannel,
	//					Version:    channelItem_.Version,
	//					VersionURL: VersionToURL(channelItem_.Version),
	//					IsCurrent:  true,
	//				})
	//			}
	//		}
	//	}
	//}

	// Add other items
	for _, group := range getGroups() {
		// TODO error handling
		_ = m.getChannelsForGroup(group, &ReleasesStatus)
	}

	return
}

func getChannelAndGroupFromVersion(releases *ReleasesStatusType, version string) (channel, group string) {
	for _, group := range getGroups() {
		for _, channel := range channelsListReverseStability {
			for _, releaseItem := range releases.Releases {
				if releaseItem.Group == group {
					for _, channelItem := range releaseItem.Channels {
						if channelItem.Name == channel {
							if channelItem.Version == version {
								return channel, group
							}
						}
					}
				}
			}
		}
	}
	return
}

// Retrieves channels and corresponding versions for the specified
// group according to the reverse order of stability
func (m *versionMenuType) getChannelsForGroup(group string, releases *ReleasesStatusType) (err error) {
	for _, item := range releases.Releases {
		if item.Group == group {
			for _, channel := range channelsListReverseStability {
				for _, channelItem := range item.Channels {
					if channelItem.Name == channel && !(channelItem.Name == m.CurrentChannel && group == m.CurrentGroup) {
						m.VersionItems = append(m.VersionItems, versionMenuItems{
							Group:      group,
							Channel:    channelItem.Name,
							Version:    channelItem.Version,
							VersionURL: VersionToURL(channelItem.Version),
							IsCurrent:  false,
						})
					}
				}
			}
		}
	}
	return
}

// Retrieves update channel groups in a descending order.
func getGroups() (groups []string) {
	for _, item := range ReleasesStatus.Releases {
		groups = append(groups, item.Group)
	}
	sort.Slice(groups, func(i, j int) bool {
		var i_, j_ float64
		var err error
		if i_, err = strconv.ParseFloat(groups[i], 32); err != nil {
			i_ = 0
		}
		if j_, err = strconv.ParseFloat(groups[j], 32); err != nil {
			j_ = 0
		}
		return i_ > j_
	})
	return
}

func updateReleasesStatus() error {
	data, err := ioutil.ReadFile("multiwerf/multiwerf.json")
	if err != nil {
		log.Printf("Can't open multiwerf.json (%e)", err)
		return err
	}
	err = json.Unmarshal(data, &ReleasesStatus)
	if err != nil {
		log.Printf("Can't unmarshall multiwerf.json (%e)", err)
		return err
	}
	return err
}

func multiwerfHandler(w http.ResponseWriter, r *http.Request) {

	// Only for test purposes
	data, err := ioutil.ReadFile("multiwerf/multiwerf.json")
	if err != nil {
		http.Error(w, "WARNING: Can't open multiwerf.json", 404)
		log.Printf("WARNING: Can't open multiwerf.json")
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = io.WriteString(w, string(data))
}

func ssiHandler(w http.ResponseWriter, r *http.Request) {
	_, _ = fmt.Fprintf(w, "<p>SSI handler (%s).</p>", r.URL.Path[1:])
	_, _ = fmt.Fprintf(w, inspectRequestHTML(r))
}

func newRouter() *mux.Router {
	r := mux.NewRouter()

	staticFileDirectoryMain := http.Dir("./root/main")
	staticFileDirectoryRu := http.Dir("./root/ru")

	r.PathPrefix("/api/status").HandlerFunc(apiStatusHandler).Methods("GET")
	r.PathPrefix("/backend/").HandlerFunc(ssiHandler).Methods("GET")
	r.PathPrefix("/documentation").HandlerFunc(documentationHandler).Methods("GET")
	r.PathPrefix("/health").HandlerFunc(healthCheckHandler).Methods("GET")
	r.Path("/includes/topnav.html").HandlerFunc(topnavHandler).Methods("GET")
	r.Path("/multiwerf").HandlerFunc(multiwerfHandler).Methods("GET")
	r.PathPrefix("/").Host("werf.io").Handler(http.FileServer(staticFileDirectoryMain))
	r.PathPrefix("/").Host("www.werf.io").Handler(http.FileServer(staticFileDirectoryMain))
	r.PathPrefix("/").Host("ng.werf.io").Handler(http.FileServer(staticFileDirectoryMain))
	r.PathPrefix("/").Host("ru.werf.io").Handler(http.FileServer(staticFileDirectoryRu))
	r.PathPrefix("/").Host("ru.ng.werf.io").Handler(http.FileServer(staticFileDirectoryRu))
	r.PathPrefix("/").Host("werf.test.flant.com").Handler(http.FileServer(staticFileDirectoryMain))
	r.PathPrefix("/").Host("werfng.test.flant.com").Handler(http.FileServer(staticFileDirectoryMain))
	r.PathPrefix("/").Host("ru.werf.test.flant.com").Handler(http.FileServer(staticFileDirectoryRu))
	r.PathPrefix("/").Host("ru.werfng.test.flant.com").Handler(http.FileServer(staticFileDirectoryRu))
	return r
}

func main() {
	r := newRouter()
	srv := &http.Server{
		Handler:      r,
		Addr:         "0.0.0.0:8080",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	log.Fatal(srv.ListenAndServe())
}
