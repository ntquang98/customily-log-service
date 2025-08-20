package main

import (
	"context"
	"log"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/template/html/v2"
	"github.com/ntquang98/shopify-log-service/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var logCollection *mongo.Collection

func main() {
	mongoURI := ""

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal(err)
	}

	logCollection = client.Database("shopify_logs").Collection("logs")

	engine := html.New("./views", ".html")
	app := fiber.New(fiber.Config{
		Views: engine,
	})

	// Add CORS middleware with full access
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "*",
		AllowMethods:     "*",
		AllowHeaders:     "*",
		AllowCredentials: false, // Set to true if you need credentials
	}))

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
	logEntry.IP = c.IP()
	logEntry.UserAgent = c.Get("User-Agent")

	ctx, cancel := context.WithTimeout(c.Context(), 10*time.Second)
	defer cancel()
	_, err := logCollection.InsertOne(ctx, logEntry)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{"status": "ok"})
}

func listLogs(c *fiber.Ctx) error {
	// query params: ?limit=20&after=<id>&before=<id>
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	after := c.Query("after", "")
	before := c.Query("before", "")

	store := c.Query("store")
	cartToken := c.Query("cart_token")
	from := c.Query("from")
	to := c.Query("to")

	filter := bson.M{}
	opts := options.Find().SetSort(bson.M{"_id": -1}).SetLimit(int64(limit))

	if store != "" {
		filter["store_domain"] = store
	}

	if cartToken != "" {
		filter["cart_token"] = cartToken
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
				timeFilter["$lte"] = t.AddDate(0, 0, 1)
			}
		}

		if len(timeFilter) > 0 {
			filter["timestamp"] = timeFilter
		}
	}

	if after != "" {
		oid, err := primitive.ObjectIDFromHex(after)
		if err == nil {
			// fetch documents with _id < after (newer → older)
			filter["_id"] = bson.M{"$lt": oid}
		}
	}

	if before != "" {
		oid, err := primitive.ObjectIDFromHex(before)
		if err == nil {
			// fetch documents with _id > before (older → newer, then reverse later)
			filter["_id"] = bson.M{"$gt": oid}
			opts.SetSort(bson.D{{Key: "_id", Value: 1}}) // ascending
		}
	}

	ctx, cancel := context.WithTimeout(c.Context(), 10*time.Second)
	defer cancel()

	// count total documents
	total, err := logCollection.CountDocuments(ctx, filter)
	if err != nil {
		return err
	}
	cur, err := logCollection.Find(ctx, filter, opts)
	if err != nil {
		return err
	}

	var logs []models.Log
	if err = cur.All(ctx, &logs); err != nil {
		return err
	}

	// if before was used, reverse so client still gets newest first
	if before != "" {
		for i, j := 0, len(logs)-1; i < j; i, j = i+1, j-1 {
			logs[i], logs[j] = logs[j], logs[i]
		}
	}

	// prepare pagination tokens
	var nextID, prevID string
	if len(logs) > 0 {
		nextID = logs[len(logs)-1].ID.Hex() // last element
		prevID = logs[0].ID.Hex()           // first element
	}

	data := fiber.Map{
		"Logs":      logs,
		"Store":     store,
		"CartToken": cartToken,
		"From":      from,
		"To":        to,
		"Limit":     limit,
		"Total":     total,
		"NextID":    nextID,
		"PrevID":    prevID,
	}

	if c.Get("HX-Request") == "true" {
		return c.Render("table", data)
	}

	return c.Render("index", data)
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
