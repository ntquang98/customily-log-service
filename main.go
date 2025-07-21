package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
	"github.com/ntquang98/shopify-log-service/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var logCollection *mongo.Collection

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatal(err)
	}

	logCollection = client.Database("shopify_logs").Collection("logs")

	engine := html.New("./views", ".html")
	engine.AddFunc("json", func(v interface{}) string {
		b, _ := json.MarshalIndent(v, "", "  ")
		return string(b)
	})
	app := fiber.New(fiber.Config{
		Views: engine,
	})

	// API
	app.Post("/log", createLog)

	// Views
	app.Get("/", listLogs)
	app.Get("/logs/:id", logDetail)

	log.Println("Listening on http://localhost:3000")
	app.Listen(":3000")
}

func createLog(c *fiber.Ctx) error {
	var logEntry models.Log

	if err := c.BodyParser(&logEntry); err != nil {
		return err
	}

	logEntry.Timestamp = time.Now()

	_, err := logCollection.InsertOne(context.Background(), logEntry)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{"status": "ok"})
}

func listLogs(c *fiber.Ctx) error {
	store := c.Query("store")
	from := c.Query("from")
	to := c.Query("to")

	filter := bson.M{}
	if store != "" {
		filter["storeDomain"] = store
	}

	if from != "" || to != "" {
		timeFilter := bson.M{}
		if from != "" {
			if t, err := time.Parse("2006-01-02", from); err == nil {
				timeFilter["$gte"] = t
			}
		}

		if to != "" {
			if t, err := time.Parse("2006-01-02", to); err == nil {
				timeFilter["$lte"] = t
			}
		}

		if len(timeFilter) > 0 {
			filter["timestamp"] = timeFilter
		}
	}

	opts := options.Find().SetSort(bson.M{"timestamp": -1}).SetLimit(50)
	cur, err := logCollection.Find(context.Background(), bson.M{}, opts)
	if err != nil {
		return err
	}

	var logs []models.Log
	if err = cur.All(context.Background(), &logs); err != nil {
		return err
	}

	if c.Get("HX-Request") == "true" {
		return c.Render("table", fiber.Map{"Logs": logs})
	}

	return c.Render("index", fiber.Map{
		"Logs":  logs,
		"Store": store,
		"From":  from,
		"To":    to,
	})
}

func logDetail(c *fiber.Ctx) error {
	id := c.Params("id")
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return c.Status(400).SendString("Invalid ID")
	}

	var logEntry models.Log
	if err := logCollection.FindOne(context.Background(), bson.M{"_id": objID}).Decode(&logEntry); err != nil {
		return c.Status(404).SendString("Log not found")
	}

	return c.Render("detail", fiber.Map{
		"Log": logEntry,
	})
}
