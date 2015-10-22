package main

import (
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"log"
	"net/http"
)

const (
	MONGODB_URL = "localhost"
	MONGO_DB    = "deviceDB"

	MONGO_COLLECTION = "device_specs"
)

type Detection struct {
	Values    []float64 `json:"values, omitempty"`
	TimeStamp float64   `json:"timestamp, omitempty"`
}

type DeviceSpecs struct {
	MAC        string       `json:"mac, omitempty"`
	TimeStamp  string       `json:"timestamp, omitempty"`
	Sensor     string       `json:"sensor, omitempty"`
	Detections []*Detection `json:"detections, omitempty"`
	Rate       float64      `json:"rate, omitempty"`
}

var (
	mDB *mgo.Database
)

func DB(mongo_url string) gin.HandlerFunc {
	session, err := mgo.Dial(mongo_url)
	if err != nil {
		panic(err)
	}

	return func(c *gin.Context) {
		s := session.Clone()
		mDB = s.DB(MONGO_DB)
		defer s.Close()

		c.Next()
	}
}

func main() {
	router := gin.Default()
	router.Use(DB(MONGODB_URL))

	router.POST("/specs", func(c *gin.Context) {
		var specs DeviceSpecs

		err := c.Bind(&specs)
		if err != nil {
			c.JSON(http.StatusBadRequest, err)
			return
		}

		detectionNum := len(specs.Detections)
		log.Printf("detection: %d\n", detectionNum)

		startTime := specs.Detections[0].TimeStamp * 1.0e-6
		stopTime := specs.Detections[detectionNum-1].TimeStamp * 1.0e-6
		log.Printf("start: %f millis\n", startTime)
		log.Printf("stop: %f millis\n", stopTime)
		log.Printf("period: %f millis\n", stopTime-startTime)

		specs.Rate = float64(detectionNum) / (stopTime - startTime)
		log.Printf("rate: %f\n mHz", specs.Rate)

		err = mDB.C(MONGO_COLLECTION).Insert(specs)
		if err != nil {
			c.JSON(http.StatusBadRequest, err)
			return
		}

		c.JSON(http.StatusCreated, specs)
	})

	router.GET("/specs", func(c *gin.Context) {
		var specs []DeviceSpecs

		err := mDB.C(MONGO_COLLECTION).Find(nil).All(&specs)
		if err != nil {
			c.JSON(http.StatusNotFound, err)
			return
		}

		c.JSON(http.StatusOK, specs)
	})

	router.DELETE("/specs", func(c *gin.Context) {
		info, err := mDB.C(MONGO_COLLECTION).RemoveAll(nil)
		if err != nil {
			c.JSON(http.StatusNotFound, err)
			return
		}

		c.JSON(http.StatusOK, info)
	})

	// router.GET("/stats/:mac", func(c *gin.Context) {
	// 	mac := c.Params.ByName("mac")

	// 	var specsList []DeviceSpecs

	// 	err := mDB.C(MONGO_COLLECTION).Find(nil).Select(bson.M{"mac": 1, "rate": 1}).Sort("rate").All(&specsList)
	// 	if err != nil {
	// 		c.JSON(http.StatusNotFound, err)
	// 		return
	// 	}

	// 	c.JSON(http.StatusOK, specsList)
	// })

	router.GET("/stat/:mac/:sensor", func(c *gin.Context) {
		mac := c.Params.ByName("mac")
		sensor := c.Params.ByName("sensor")

		var specsList []DeviceSpecs

		err := mDB.C(MONGO_COLLECTION).Find(bson.M{"sensor": sensor}).Select(bson.M{"mac": 1, "rate": 1}).Sort("-rate").All(&specsList)
		if err != nil {
			c.JSON(http.StatusNotFound, err)
			return
		}

		for ranking, specs := range specsList {
			if mac == specs.MAC {
				c.JSON(http.StatusOK, bson.M{"ranking": ranking + 1, "rate": specs.Rate})
				return
			}
		}

		c.JSON(http.StatusNotFound, nil)
	})

	router.Run(":8080")
}
