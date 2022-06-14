package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/OdyseeTeam/e-mage/config"
	http "github.com/OdyseeTeam/e-mage/server"
	"github.com/OdyseeTeam/gody-cdn/configs"
	"github.com/OdyseeTeam/gody-cdn/store"
	"github.com/OdyseeTeam/mirage/optimizer"

	"github.com/lbryio/lbry.go/v2/extras/errors"
	"github.com/lbryio/lbry.go/v2/extras/stop"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	//Bind to Viper
	err := viper.BindPFlags(serveCmd.PersistentFlags())
	if err != nil {
		logrus.Panic(err)
	}
	rootCmd.AddCommand(serveCmd)
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Runs the e-mage server",
	Long:  `Runs the e-mage server`,
	Args:  cobra.OnlyValidArgs,
	Run: func(cmd *cobra.Command, args []string) {
		stopper := stop.New()
		config.InitializeConfiguration()
		err := configs.Init("config.json")
		if err != nil {
			logrus.Fatalln(errors.FullTrace(err))
		}
		s3Store := store.NewS3Store(configs.Configuration.S3Origins[0])
		localDB := configs.Configuration.LocalDB
		localDsn := fmt.Sprintf("%s:%s@tcp(%s:3306)/%s", localDB.User, localDB.Password, localDB.Host, localDB.Database)
		dbs := store.NewDBBackedStore(s3Store, localDsn)
		httpServer := http.NewServer(optimizer.NewOptimizer(), dbs)
		err = httpServer.Start(":6456")
		if err != nil {
			logrus.Fatal(err)
		}
		defer httpServer.Shutdown()

		interruptChan := make(chan os.Signal, 1)
		signal.Notify(interruptChan, os.Interrupt, syscall.SIGTERM)
		<-interruptChan
		stopper.StopAndWait()
	},
}
