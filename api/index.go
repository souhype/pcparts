package handler

import (
	"database/sql"
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed templates/*
var t embed.FS

//go:embed db.sqlite
var dbFile embed.FS

type Page struct {
	Products  []*Product
	Timestamp string
	Count     uint16
	Index     uint16
}

type Product struct {
	ID        uint16
	Name      string
	Url       string
	ImgUrl    string
	Title     string
	Price     float32
	UsedPrice float32
	Save      float32
	Region    string
	Timestamp string
}

type QueryParams struct {
	Queries    []string
	Offset     uint16
	OrderBy    string
	StartIndex uint16
}

func getProductsBySearch(params QueryParams) (Page, error) {
	tempFile, err := os.CreateTemp("", "db-*.sqlite")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(tempFile.Name())

	// Write the embedded database to the temporary file
	dbData, err := dbFile.ReadFile("db.sqlite")
	if err != nil {
		log.Fatal(err)
	}
	_, err = tempFile.Write(dbData)
	if err != nil {
		log.Fatal(err)
	}
	tempFile.Close()

	db, err := sql.Open("sqlite3", tempFile.Name())
	if err != nil {
		fmt.Println(err)
	}
	defer db.Close()

	queryString, queryParams := buildQuery(params)
	fmt.Println(queryString)

	stmt, err := db.Prepare(queryString)
	if err != nil {
		fmt.Println(err)
	}
	rows, err := stmt.Query(queryParams...)
	if err != nil {
		fmt.Println(err)
	}

	products := make([]*Product, 0, 20)
	index := params.StartIndex
	for rows.Next() {
		product := new(Product)
		err := rows.Scan(&product.ID, &product.Name, &product.Url, &product.ImgUrl, &product.Title, &product.Price, &product.UsedPrice, &product.Save, &product.Region, &product.Timestamp)
		if err != nil {
			log.Fatal(err)
		}
		product.ID = index + 1
		products = append(products, product)
		index++
	}

	count, err := getCount(db)
	if err != nil {
		fmt.Println(err)
	}

	timestamp, err := getLatestTimestamp(db)
	if err != nil {
		fmt.Println(err)
	}

	return Page{
		Products:  products,
		Timestamp: timestamp,
		Count:     count,
		Index:     uint16(index),
	}, nil
}

func buildQuery(params QueryParams) (string, []interface{}) {
	conditions := []string{}
	queryParams := []interface{}{}
	baseQuery := "SELECT * from products WHERE "

	for i, query := range params.Queries {
		conditions = append(conditions, fmt.Sprintf("name LIKE $%d", i+1))
		queryParams = append(queryParams, "%"+query+"%")
	}
	queryString := baseQuery + strings.Join(conditions, " AND ") + fmt.Sprintf(" ORDER BY %s LIMIT 20 OFFSET $%d", params.OrderBy, len(queryParams)+1)
	queryParams = append(queryParams, params.Offset)

	fmt.Println("BUILD Q "+queryString, queryParams)
	return queryString, queryParams
}

func getCount(db *sql.DB) (uint16, error) {
	var count uint16
	countQuery := "SELECT COUNT(*) FROM products"
	err := db.QueryRow(countQuery).Scan(&count)
	fmt.Println(err)
	return count, err
}

func getLatestTimestamp(db *sql.DB) (string, error) {
	var timestamp string
	timestampQuery := "SELECT timestamp FROM products ORDER BY timestamp DESC LIMIT 1"
	err := db.QueryRow(timestampQuery).Scan(&timestamp)
	if err != nil {
		fmt.Println(err)
	}
	t, _ := time.Parse(time.RFC3339, timestamp)
	return t.Format("02-Jan-2006 15:04:05"), nil
}

func Handler(w http.ResponseWriter, r *http.Request) {
	tmpl, _ := template.ParseFS(t, "templates/*")
	templateName := r.URL.Query().Get("template")
	queries := strings.Split(r.URL.Query().Get("search"), " ")
	order := r.URL.Query().Get("order")
	if order == "" {
		order = "price ASC"
	}
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	fmt.Println(order)

	if templateName == "" {
		templateName = "index"
	}
	startIndex := uint16(offset)

	products, _ := getProductsBySearch(QueryParams{Queries: queries, Offset: uint16(offset), StartIndex: startIndex, OrderBy: order})

	w.Header().Set("Content-Type", "text/html")

	tmpl.ExecuteTemplate(w, templateName, Page{Products: products.Products, Count: products.Count, Timestamp: products.Timestamp})
}
