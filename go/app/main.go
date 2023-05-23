package main

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	_ "github.com/mattn/go-sqlite3"
)

var DbConnection *sql.DB

const (
	ImgDir = "images"
)

type Response struct {
	Message string `json:"message"`
}

type Items struct {
	Items []Item `json:"items"`
}

type Item struct {
	Id       string `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category"`
	Image    string `json:"image"`
}

func root(c echo.Context) error {
	res := Response{Message: "Hello, world!"}
	
	return c.JSON(http.StatusOK, res)
}

/* 3.4
curl -X POST --url http://localhost:9000/items -F name=jacket -F category=fashion -F image=@./local_image.jpg
*/

// ハッシュ化
func getSHA256Binary(s string) []byte {
	r := sha256.Sum256([]byte(s))
	return r[:]
}

//dbが存在しなかったときに作る
func makeDB(){
	// データベースを開く。なければ生成
	DbConnection, err := sql.Open("sqlite3", "../db/mercari.sqlite3")
	
	cmd := `CREATE TABLE IF NOT EXISTS category (
		id integer primary key autoincrement,
		name string NOT NULL
		);
		CREATE TABLE IF NOT EXISTS items(
        id integer primary key autoincrement, 
		name string NOT NULL, 
		category_id integer NOT NULL, 
		image_name string,
		FOREIGN KEY (category_id) REFERENCES Category(id)
		);`

	//実行 結果は返ってこない為、_にする
	_, err = DbConnection.Exec(cmd)

	//エラーハンドリング
	if err != nil {
		log.Fatal(err)
	}
	//閉じる
	defer DbConnection.Close()
}


func addItem(c echo.Context) error {
	// Get form data
	name := c.FormValue("name")
	category := c.FormValue("category")
	image, err := c.FormFile("image")
	if err != nil {
		return err
	}

	//hashしたものに".jpg"をつける
	hashedImg := getSHA256Binary(image.Filename)
	image_filename := fmt.Sprintf("%x%s", hashedImg, ".jpg")

	c.Logger().Infof("Receive item: %s", name)
	c.Logger().Infof("Receive item: %s", category)
	c.Logger().Infof("Receive item: %s", image.Filename)

	// 新しいitemを作る
	var newItem Item
	newItem.Name = name
	newItem.Category = category
	newItem.Image = image_filename

	// makeDB()

	DbConnection, err := sql.Open("sqlite3", "../db/mercari.sqlite3")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err)
	}
	defer DbConnection.Close()

	// 追加
	var cmd string
	
	//categoryに追加
	var categoryId int64
	c.Logger().Infof(category)
	err = DbConnection.QueryRow("SELECT id FROM category WHERE name = ?", category).Scan(&categoryId)
	if err != nil {
		c.Logger().Infof("カテゴリー無し")
		
		DbConnection, err := sql.Open("sqlite3", "../db/mercari.sqlite3")
		
		cmd := "INSERT INTO category (name) VALUES (?)"
		add, err := DbConnection.Exec(cmd, category)

		if err != nil {
			c.Logger().Infof("エラー", err)
		}
		categoryId, err = add.LastInsertId()
		c.Logger().Infof("categotyId is: %s", categoryId)
		if err != nil {
			log.Fatal(err)
		}
	}
	// VALUES (?, ?)  値は後で渡す。セキュリテイの関係でこのようにする方がいいらしい
	cmd = "INSERT INTO items (name, category_id, image_name) VALUES (?, ?, ?)"
	_, err = DbConnection.Exec(cmd, newItem.Name, categoryId, newItem.Image)
	if err != nil {
		c.Logger().Infof("エラー", err)
	}

	message := fmt.Sprintf("item received: %s", name)
	res := Response{Message: message}

	return c.JSON(http.StatusOK, res)
}

func ReadDB() ([]Item, error) {
	//Read
	cmd := "SELECT items.id, items.name, category.name, items.image_name FROM items JOIN category ON items.category_id = category.id"
	rows, err := DbConnection.Query(cmd)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	item := []Item{}
	for rows.Next() {
		var element Item
	  	err := rows.Scan(&element.Id, &element.Name, &element.Category, &element.Image)
	  	if err != nil {
		return nil, err
		}
	  	item = append(item, element)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return item, nil
}

func getItem(c echo.Context) error {

	// makeDB()
	DbConnection, err := sql.Open("sqlite3", "../db/mercari.sqlite3")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err)
	}
	// item, _ := ReadDB()
	defer DbConnection.Close()

	//Read
	cmd := "SELECT items.id, items.name, category.name, items.image_name FROM items JOIN category ON items.category_id = category.id"
	rows, _ := DbConnection.Query(cmd)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err)
	}
	// item, _ := ReadDB()
	defer rows.Close()

	// 取得したデータをループでスライスに追加　for rows.Next()
	item := []Item{}
	for rows.Next() {
		var element Item
		//scan データ追加
		err := rows.Scan(&element.Id, &element.Name, &element.Category, &element.Image)
		if err != nil {
			c.Logger().Infof("sqlの中身にエラー", err)
		}
		item = append(item, element)
	}

	err = rows.Err()
	if err != nil {
		c.Logger().Infof("エラー", err)
	}
	// item, _ := ReadDB()

	// naosu item
	return c.JSON(http.StatusOK, item)
}

func getImg(c echo.Context) error {
	// Create image path
	imgPath := path.Join(ImgDir, c.Param("imageFilename"))

	if !strings.HasSuffix(imgPath, ".jpg") {
		res := Response{Message: "Image path does not end with .jpg"}
		return c.JSON(http.StatusBadRequest, res)
	}
	if _, err := os.Stat(imgPath); err != nil {
		c.Logger().Debugf("Image not found: %s", imgPath)
		imgPath = path.Join(ImgDir, "default.jpg")
	}
	return c.File(imgPath)
}

func getItemByID(c echo.Context) error {
	// makeDB()
	DbConnection, err := sql.Open("sqlite3", "../db/mercari.sqlite3")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err)
	}
	defer DbConnection.Close()
	//Read
	cmd := "SELECT items.id, items.name, category.name, items.image_name FROM items JOIN category ON items.category_id = category.id"
	rows, _ := DbConnection.Query(cmd)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err)
	}
	defer rows.Close()

	// 取得したデータをループでスライスに追加　for rows.Next()
	item := []Item{}
	for rows.Next() {
		var element Item
		//scan データ追加
		err := rows.Scan(&element.Id, &element.Name, &element.Category, &element.Image)
		if err != nil {
			c.Logger().Infof("sqlの中身にエラー", err)
		}
		item = append(item, element)
	}

	err = rows.Err()
	if err != nil {
		c.Logger().Infof("エラー", err)
	}

	//表示
	for _, element := range item {
		fmt.Println(element.Name, element.Category)
	}

	err = rows.Err()
	if err != nil {
		c.Logger().Infof("エラー", err)
	}

	id := c.Param("item_id")
	c.Logger().Infof("Receive item: %s", id)
	c.Logger().Infof("&&&&&&&Receive item: %s", item)
	for i, ele := range item {
		var j = i+1
		if strconv.Itoa(j) == id {
			return c.JSON(http.StatusOK, ele)
		}
	}
	res := Response{Message: "Not found"}
	return c.JSON(http.StatusNotFound, res)
}

func searchItems(c echo.Context) error {
	// makeDB()
	DbConnection, err := sql.Open("sqlite3", "../db/mercari.sqlite3")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err)
	}
	defer DbConnection.Close()
	//Read
	cmd := "SELECT items.id, items.name, category.name, items.image_name FROM items JOIN category ON items.category_id = category.id"
	rows, _ := DbConnection.Query(cmd)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err)
	}
	defer rows.Close()

	// 取得したデータをループでスライスに追加　for rows.Next()
	item := []Item{}
	for rows.Next() {
		var element Item
		//scan データ追加
		err := rows.Scan(&element.Id, &element.Name, &element.Category, &element.Image)
		if err != nil {
			c.Logger().Infof("sqlの中身にエラー", err)
		}
		item = append(item, element)
	}

	err = rows.Err()
	if err != nil {
		c.Logger().Infof("エラー", err)
	}

	//表示
	for _, element := range item {
		fmt.Println(element.Name, element.Category)
	}

	var keyword string = c.QueryParam("keyword")
	var searchedItem []Item
	for _, ele := range item {
		if keyword == ele.Category {
			searchedItem = append(searchedItem, ele)
		}
	}
	if len(searchedItem) != 0 {
		return c.JSON(http.StatusNotFound, searchedItem)
	}
	res := Response{Message: "Not found"}
	return c.JSON(http.StatusNotFound, res)
}

func main() {
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Logger.SetLevel(log.INFO)

	front_url := os.Getenv("FRONT_URL")
	if front_url == "" {
		front_url = "http://localhost:3000"
	}
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{front_url},
		AllowMethods: []string{http.MethodGet, http.MethodPut, http.MethodPost, http.MethodDelete},
	}))

	// Routes
	e.GET("/", root)
	e.POST("/items", addItem)
	e.GET("/items", getItem)
	e.GET("/image/:imageFilename", getImg)
	e.GET("/items/:item_id", getItemByID)
	e.GET("/search", searchItems)
	// Start server
	e.Logger.Fatal(e.Start(":9000"))
}