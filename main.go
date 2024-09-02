package main

import (
	"context"
	"crypto/rand"
	"errors"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	logger "github.com/sirupsen/logrus"
)

func main() {
	databaseClient, database := InitDatabaseConnection()
	ctx := context.Background()

	if err := databaseClient.Ping(ctx, readpref.Primary()); err != nil {
		logger.Fatalf("Failed to determine primary database server: %v", err)
	}
	logger.Info("Successfully connected to database")

	DBHelper = NewDatabaseHelper(databaseClient.Database(database))

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGKILL, syscall.SIGINT)
	var nonSecureWebServer *echo.Echo
	var webServer *echo.Echo
	go func() {
		s := <-signals
		logger.Warnf("Shutting down because received signal: %s", s)
		if err := nonSecureWebServer.Shutdown(ctx); err != nil {
			logger.Warnf("Problem with shutting down non secure server: %v", err)
		}
		if err := webServer.Shutdown(ctx); err != nil {
			logger.Warnf("Problem with shutting down secure server: %v", err)
		}
		os.Exit(0)
	}()

	IsConfigurationMode()

	nonSecureWebServer = echo.New()
	nonSecureWebServer.StdLogger = RuntimeLogger
	nonSecureWebServer.Logger.SetOutput(RuntimeLogger.Writer())
	nonSecureWebServer.Pre(middleware.HTTPSRedirect())
	go func() {
		if err := nonSecureWebServer.Start(":80"); err != nil {
			logger.Fatalf("Cannot listen on port :80 : %v", err)
		}
	}()

	webServer = echo.New()

	// Middleware: secure
	webServer.Use(middleware.Secure())

	// Middleware: mongodb session store
	// Values:
	// 	- id: User ID (objectID.Hex())
	// 	- login: User login (string)
	//	- filters: User filters (UserSavedFilters)
	//	- admin: Admin logged in (bool)
	//	- profile: User profile (UserProfileStruct)
	var secureSessionKey DataSecureSessionKey
	filter := BsonExists("secure_session_key")
	if err := DBHelper.FindOne(DataCollection, filter, &secureSessionKey); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			key := make([]byte, 32)
			if _, err := rand.Read(key); err != nil {
				logger.Fatalf("Failed to generate secure key: %v", err)
			}
			secureSessionKey = DataSecureSessionKey{SecureSessionKey: key}
			if err := DBHelper.InsertOne(DataCollection, &secureSessionKey); err != nil {
				logger.Fatalf("Failed to insert new secure session key: %v", err)
			}
		} else {
			logger.Fatalf("Failed to get secure session key: %v", err)
		}
	}
	webServer.Use(session.Middleware(NewHttpSessionMongoDB(databaseClient.Database(database).Collection(SessionsCollection), HttpSessionDurationSeconds, secureSessionKey.SecureSessionKey)))

	// Middleware: gzip
	webServer.Use(middleware.GzipWithConfig(middleware.GzipConfig{
		Level: 9,
	}))

	// Handle not found errors
	webServer.HTTPErrorHandler = func(err error, c echo.Context) {
		rcvCode := http.StatusInternalServerError
		c.Logger().Warnf("Error encountered during request: %v", err)
		var he *echo.HTTPError
		if errors.As(err, &he) {
			rcvCode = he.Code
		}
		if rcvCode == http.StatusNotFound {
			if c.Request().Header.Get("X-Requested-With") == "xmlhttprequest" {
				if err := c.NoContent(http.StatusNotFound); err != nil {
					c.Logger().Warnf("Failed to send status not found error: %v", err)
				}
			} else {
				if err := c.Redirect(http.StatusMovedPermanently, "http://localhost:3000/notFound"); err != nil {
					c.Logger().Warnf("Failed to perform redirect to not found page: %v", err)
				}
			}
		} else {
			if err := c.NoContent(rcvCode); err != nil {
				c.Logger().Warnf("Failed to send error code: %v", err)
			}
		}
	}

	webServerGroup := webServer.Group("")
	webServerGroup.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:  []string{"http://localhost:3000"},
		AllowMethods:  []string{echo.GET, echo.PUT, echo.POST, echo.DELETE},
		AllowHeaders:  []string{"*"},
		ExposeHeaders: []string{echo.HeaderContentType},
	}))

	webServerGroup.GET("/isInEditMode", func(c echo.Context) error {
		IsConfigurationMode()
		return c.JSON(http.StatusOK, ConfigurationMode)
	})

	webServerGroup.POST("/addKubeconfig", func(c echo.Context) error {
		kubeconfig, err := c.FormFile("kubeconfig")
		if err != nil {
			logger.Warnf("Failed to get kubeconfig when calling AddKubeconfigHandler: %v", err)
			return c.NoContent(http.StatusInternalServerError)
		}
		handler := &AddKubeconfigHandler{name: c.FormValue("name"), kubeconfig: kubeconfig}
		return handler.ServeHTTP(c)
	})

	webServerGroup.POST("/addKubeconfigText", func(c echo.Context) error {
		kubeconfig := c.FormValue("kubeconfig")
		handler := &AddKubeconfigTextHandler{name: c.FormValue("name"), kubeconfig: kubeconfig}
		return handler.ServeHTTP(c)
	})

	webServerGroup.POST("/scaleDeployment/:id/:name/:ns/:deployment", func(c echo.Context) error {
		handler := &ScaleDeploymentHandler{
			ID:         c.Param("id"),
			Name:       c.Param("name"),
			NS:         c.Param("ns"),
			Deployment: c.Param("deployment"),
		}
		return handler.ServeHTTP(c)
	})

	webServerGroup.GET("/lac", LogActivityConsole)

	webServerGroup.GET("/getKubeconfigs", func(c echo.Context) error {
		handler := &GetKubeconfigsHandler{}
		return handler.ServeHTTP(c)
	})

	webServerGroup.GET("/getK8sClustersNSs/:id/:name", func(c echo.Context) error {
		handler := &GetK8sClusterNSsHandler{
			ID:   c.Param("id"),
			Name: c.Param("name"),
		}
		return handler.ServeHTTP(c)
	})

	webServerGroup.GET("/getK8sdeployments/:id/:name/:ns", func(c echo.Context) error {
		handler := &GetK8sDeploymentsHandler{
			ID:   c.Param("id"),
			Name: c.Param("name"),
			NS:   c.Param("ns"),
		}
		return handler.ServeHTTP(c)
	})

	webServerGroup.GET("/getK8sdeploymentInfo/:id/:name/:ns/:deployment", func(c echo.Context) error {
		handler := &GetK8sDeploymentInfoHandler{
			ID:         c.Param("id"),
			Name:       c.Param("name"),
			NS:         c.Param("ns"),
			Deployment: c.Param("deployment"),
		}
		return handler.ServeHTTP(c)
	})

	webServerGroup.GET("/getK8sstateFulSets/:id/:name/:ns", func(c echo.Context) error {
		handler := &GetK8sStateFulSetsHandler{
			ID:   c.Param("id"),
			Name: c.Param("name"),
			NS:   c.Param("ns"),
		}
		return handler.ServeHTTP(c)
	})

	webServerGroup.GET("/getK8sdaemonSets/:id/:name/:ns", func(c echo.Context) error {
		handler := &GetK8sDaemonSetsHandler{
			ID:   c.Param("id"),
			Name: c.Param("name"),
			NS:   c.Param("ns"),
		}
		return handler.ServeHTTP(c)
	})

	webServerGroup.GET("/getK8sjobs/:id/:name/:ns", func(c echo.Context) error {
		handler := &GetK8sJobsHandler{
			ID:   c.Param("id"),
			Name: c.Param("name"),
			NS:   c.Param("ns"),
		}
		return handler.ServeHTTP(c)
	})

	webServerGroup.GET("/getK8scronJobs/:id/:name/:ns", func(c echo.Context) error {
		handler := &GetK8sCronJobsHandler{
			ID:   c.Param("id"),
			Name: c.Param("name"),
			NS:   c.Param("ns"),
		}
		return handler.ServeHTTP(c)
	})

	webServerGroup.GET("/getK8spods/:id/:name/:ns", func(c echo.Context) error {
		handler := &GetK8sPodsHandler{
			ID:   c.Param("id"),
			Name: c.Param("name"),
			NS:   c.Param("ns"),
		}
		return handler.ServeHTTP(c)
	})

	webServerGroup.GET("/getK8sreplicaSets/:id/:name/:ns", func(c echo.Context) error {
		handler := &GetK8sReplicaSetsHandler{
			ID:   c.Param("id"),
			Name: c.Param("name"),
			NS:   c.Param("ns"),
		}
		return handler.ServeHTTP(c)
	})

	webServerGroup.GET("/getK8sreplicaControllers/:id/:name/:ns", func(c echo.Context) error {
		handler := &GetK8sReplicaControllersHandler{
			ID:   c.Param("id"),
			Name: c.Param("name"),
			NS:   c.Param("ns"),
		}
		return handler.ServeHTTP(c)
	})

	webServerGroup.DELETE("/deleteKubeconfig/:id", func(c echo.Context) error {
		handler := &DeleteKubeconfigHandler{
			ID: c.Param("id"),
		}
		return handler.ServeHTTP(c)
	})

	// Listen on 443 port
	certFile := CurrentDirectory + PathSeparator + "certs" + PathSeparator + "server.cert"
	keyFile := CurrentDirectory + PathSeparator + "certs" + PathSeparator + "server.key"
	webServer.StdLogger = RuntimeLogger
	webServer.Logger.SetOutput(RuntimeLogger.Writer())
	if err := webServer.StartTLS(":443", certFile, keyFile); err != nil {
		logger.Fatalf("Cannot listen on port :443 : %v", err)
	}
}
