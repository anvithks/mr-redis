package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/mesos/mr-redis/common/types"
	"github.com/mesos/mr-redis/sched/cmd"
	"github.com/mesos/mr-redis/sched/httplib"
	"github.com/mesos/mr-redis/sched/mesoslib"
)

//Declare all the Constants to be used in this file
const (
	//HTTP_SERVER_PORT Rest server of the scheduler by default
	HTTP_SERVER_PORT = "8080"
)

//MrRedisConfig struct of the json config file that is used while starting the scheduler
type MrRedisConfig struct {
	UserName      string //Supply a username
	FrameworkName string //Supply a frameworkname
	Master        string //MesosMaster's endpoint zk://mesos.master/2181 or 10.11.12.13:5050
	ExecutorPath  string //Executor's Path from where to distribute
	RedisImage    string //Redis Image should be downloaded
	DBType        string //Type of the database etcd/zk/mysql/consul etcd.,
	DBEndPoint    string //Endpoint of the database
	LogFile       string //Name of the logfile
	ArtifactIP    string //The IP to which we should bind to for distributing the executor among the interfaces
	ArtifactPort  string //The port to which we should bind to for distributing the executor
	HTTPPort      string //Defaults to 8080 if otherwise specify explicitly
}

//NewMrRedisDefaultConfig Default Constructor to create a config file
func NewMrRedisDefaultConfig() MrRedisConfig {
	return MrRedisConfig{
		UserName:      "ubuntu",
		FrameworkName: "MrRedis",
		Master:        "127.0.0.1:5050",
		ExecutorPath:  "./MrRedisExecutor",
		RedisImage:    "redis:3.0-alpine",
		DBType:        "etcd",
		DBEndPoint:    "127.0.0.1:2379",
		LogFile:       "stderr",
		ArtifactIP:    "127.0.0.1",
		ArtifactPort:  "5454",
		HTTPPort:      "5656",
	}
}

func main() {

	cfgFileName := flag.String("config", "./config.json", "Supply the location of MrRedis configuration file")
	dumpConfig := flag.Bool("DumpEmptyConfig", false, "Dump Empty Config file")
	flag.Parse()

	Cfg := NewMrRedisDefaultConfig()

	if *dumpConfig == true {
		configBytes, err := json.MarshalIndent(Cfg, " ", "  ")
		if err != nil {
			log.Printf("Error marshalling the dummy config file. Exiting %v", err)
			return
		}
		fmt.Printf("%s\n", string(configBytes))
		return
	}

	cfgFile, err := ioutil.ReadFile(*cfgFileName)

	if err != nil {
		log.Printf("Error Reading the configration file. Resorting to default values")
	}
	err = json.Unmarshal(cfgFile, &Cfg)
	if err != nil {
		log.Fatalf("Error parsing the config file %v", err)
	}
	log.Printf("Configuration file is = %v", Cfg)

	log.Printf("*****************************************************************")
	log.Printf("*********************Starting MrRedis-Scheduler******************")
	log.Printf("*****************************************************************")
	//Command line argument parsing

	//Facility to overwrite the etcd endpoint for scheduler if its running in the same docker container and expose a different one for executors

	dbEndpoint := os.Getenv("ETCD_LOCAL_ENDPOINT")

	if dbEndpoint == "" {
		dbEndpoint = Cfg.DBEndPoint
	}

	//Initalize the common entities like store, store configuration etc.
	isInit, err := types.Initialize(Cfg.DBType, dbEndpoint)
	if err != nil || isInit != true {
		log.Fatalf("Failed to intialize Error:%v return %v", err, isInit)
	}

	//Start the Mesos library
	go mesoslib.Run(Cfg.Master, Cfg.ArtifactIP, Cfg.ArtifactPort, Cfg.ExecutorPath, Cfg.RedisImage, Cfg.DBType, Cfg.DBEndPoint, Cfg.FrameworkName, Cfg.UserName)

	//Start the creator
	go cmd.Creator()

	//Start the Mainterainer
	go cmd.Maintainer()

	//Start the Destroyer
	go cmd.Destoryer()

	//Start HTTP server and related things to handle restfull calls to the scheduler
	httplib.Run(Cfg.HTTPPort)

	//Wait for termination signal

	log.Printf("*****************************************************************")
	log.Printf("*********************Finished MrRedis-Scheduler******************")
	log.Printf("*****************************************************************")

}
