package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	ldap "gopkg.in/ldap.v2"

	"github.com/compsoc-edinburgh/bi-provider/pkg/config"
	"github.com/gin-gonic/gin"
	"github.com/qaisjp/gosign"
	"github.com/sirupsen/logrus"
)

var outNotLoggedIn = gin.H{
	"status":  "error",
	"message": "not logged in",
}

// NewAPI sets up a new API module.
func NewAPI(
	conf *config.Config,
	log *logrus.Logger,
) *API {

	router := gin.Default()

	// security measures
	router.Use(
		func(c *gin.Context) {
			// Grant access to either betterinformatics.com, or alpha.betterinformatics.com, but no other website.
			origin := c.Request.Header.Get("Origin")
			if (origin == "https://betterinformatics.com") || (origin == "https://alpha.betterinformatics.com") {
				c.Header("Access-Control-Allow-Origin", origin)
			} else {
				c.Header("Access-Control-Allow-Origin", "https://betterinformatics.com")
			}

			c.Header("Vary", "Origin, Cookie")
			c.Header("Cache-Control", "max-age=3600")
			c.Header("X-Frame-Options", "DENY")
			c.Header("Content-Type", "application/json")
			c.Header("Access-Control-Allow-Credentials", "true")

			c.Next()
		},
	)

	a := &API{
		Config: conf,
		Log:    log,
		Gin:    router,
	}

	router.GET("/", a.provide)

	return a
}

func (a *API) provide(c *gin.Context) {
	cookie, err := c.Cookie("cosign-betterinformatics.com")
	if err != nil {
		c.JSON(http.StatusUnauthorized, outNotLoggedIn)

		return
	}

	url := "http://localhost:6663/check" +
		"/" + a.Config.CoSign.Name +
		"/" + a.Config.CoSign.Password +
		"?ip=" + c.ClientIP() +
		"&cookie=" + strings.Replace(cookie, " ", "%2B", -1)

	resp, err := http.Get(url)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	// defer resp.Body.Close()
	// contents, err := ioutil.ReadAll(resp.Body)
	// if err != nil {
	// 	fmt.Printf("%s", err)
	// 	os.Exit(1)
	// }
	// fmt.Printf("%s\n", contents)

	decoder := json.NewDecoder(resp.Body)
	defer resp.Body.Close()

	var result struct {
		Status  string
		Message string
		Data    gosign.CheckResponse
	}
	err = decoder.Decode(&result)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "could not decode JSON",
		})
		return
	}

	if resp.StatusCode == http.StatusUnauthorized {
		c.JSON(http.StatusUnauthorized, outNotLoggedIn)
		return
	}

	if result.Status != "success" {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "cosign-webapi: " + result.Message,
		})
		return
	}

	if result.Data.Realm != "INF.ED.AC.UK" {
		c.JSON(http.StatusForbidden, gin.H{
			"status":  "error",
			"message": "Access denied. Realm " + result.Data.Realm + " is not permitted.",
		})

		return
	}

	out, err := getGroups(result.Data.Principal)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "ldap: " + err.Error(),
		})

		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   out,
	})
}

type outStruct struct {
	Username  string
	Name      string
	Year      string
	Modules   []string
	Degree    string
	Cohort    string
	IsStudent bool
}

func getGroups(u string) (out outStruct, err error) {
	conn, err := ldap.Dial("tcp", "localhost:1389")
	if err != nil {
		return out, err
	}
	defer conn.Close()

	out.Username = u

	// Build the search request for retrieving their name
	searchRequest := ldap.NewSearchRequest(
		"dc=inf,dc=ed,dc=ac,dc=uk",
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases,
		0, 0, false,
		fmt.Sprintf("(uid=%s)", u), []string{"givenName"},
		nil,
	)

	// Search for their name
	nr, err := conn.Search(searchRequest)
	if err != nil {
		return out, err
	}

	// Retrieve their name
	for _, entry := range nr.Entries {
		out.Name = entry.GetAttributeValue("givenName")
	}

	// Build the search request for retrieving their groups
	searchRequest = ldap.NewSearchRequest(
		"dc=inf,dc=ed,dc=ac,dc=uk", // BaseDN

		ldap.ScopeWholeSubtree, // default for ldapsearch
		ldap.NeverDerefAliases, // default for ldapsearch
		0, 0, false,
		fmt.Sprintf("(member=uid=%s,ou=People,dc=inf,dc=ed,dc=ac,dc=uk)", u),
		nil, // this means ALL  -- just requesting `cn`` is ok //[]string{"cn"},
		nil,
	)

	// Search for their groups... (via capabilities)
	sr, err := conn.Search(searchRequest)
	if err != nil {
		return out, err
	}

	// Retrieve the entries and store them in the output struct
	for _, entry := range sr.Entries {
		group := entry.GetAttributeValue("cn")
		if group == "role/student" {
			out.IsStudent = true
		}

		if strings.HasPrefix(group, "role/year-") {
			out.Year = group[10:]
		}

		if strings.HasPrefix(group, "role/degree-") {
			out.Degree = group[12:]
		}

		if strings.HasPrefix(group, "role/cohort-") {
			out.Cohort = group[12:]
		}

		if strings.HasPrefix(group, "role/module-") {
			out.Modules = append(out.Modules, group[12:])
		}
	}

	if out.IsStudent {
		// Forum > FH
		// ... but where does AT lie?
		//
		out.IsStudent = out.Cohort != "pgr"
	}

	return out, nil
}
