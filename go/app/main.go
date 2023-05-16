package main

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"
	"crypto/sha256"
	"strconv"
	"database/sql"

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
	Id string `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category"`
	Image string `json:"image"`
}

func root(c echo.Context) error {
	res := Response{Message: "Hello, world!"}
	return c.JSON(http.StatusOK, res)
}

/* 3.4 
curl -X POST 
--url http://localhost:9000/items 
-F name=jacket 
-F category=fashion 
-F image=@./local_image.jpg
*/

//ハッシュ化
func getSHA256Binary(s string) []byte {
    r := sha256.Sum256([]byte(s))
    return r[:]
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
	image_filename := fmt.Sprintf("%x%s",hashedImg , ".jpg")

	c.Logger().Infof("Receive item: %s", name)
	c.Logger().Infof("Receive item: %s", category)
	c.Logger().Infof("Receive item: %s", image.Filename)

	// 新しいitemを作る
	var newItem Item
	newItem.Name = name
	newItem.Category = category
	newItem.Image = image_filename
	
	// データベースを開く。なければ生成
	DbConnection, _ := sql.Open("sqlite3", "../../db/items.db")
	//閉じる
	defer DbConnection.Close()

	cmd := `CREATE TABLE IF NOT EXISTS items(
        id int primary key autoincrement, 
		name string, 
		category string, 
		image_name string)`

	//実行 結果は返ってこない為、_にする
    _, err = DbConnection.Exec(cmd)

    //エラーハンドリング
    if err != nil {
        c.Logger().Infof("エラー", err)
    }
	

	// 追加
	// VALUES (?, ?)  値は後で渡す。セキュリテイの関係でこのようにする方がいいらしい
    cmd = "INSERT INTO items (name, category, image_name) VALUES (?, ?, ?, ?)"
	_, err = DbConnection.Exec(cmd, newItem.Name, newItem.Category, newItem.Image)
    if err != nil {
        c.Logger().Infof("エラー", err)
    }

	message := fmt.Sprintf("item received: %s", name)
	res := Response{Message: message}

	return c.JSON(http.StatusOK, res)
}

func getItem(c echo.Context) error {

	// データベースを開く。なければ生成
	DbConnection, _ := sql.Open("sqlite3", "../../db/items.db")
	//閉じる
	defer DbConnection.Close()

	cmd := `CREATE TABLE IF NOT EXISTS items(
        id int primary key autoincrement, 
		name string, 
		category string, 
		image_name string)`

	//実行 結果は返ってこない為、_にする
    _, err := DbConnection.Exec(cmd)

    //エラーハンドリング
    if err != nil {
        c.Logger().Infof("エラー", err)
    }
	
	//Read
	cmd = "SELECT * FROM person"
    rows, _ := DbConnection.Query(cmd)
    defer rows.Close()
	
	// 取得したデータをループでスライスに追加　for rows.Next()
    var item []Item
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


func getItemByID(c echo.Context)  error{
	// データベースを開く。なければ生成
	DbConnection, _ := sql.Open("sqlite3", "../../db/items.db")
	//閉じる
	defer DbConnection.Close()

	cmd := `CREATE TABLE IF NOT EXISTS items(
        id int primary key autoincrement, 
		name string, 
		category string, 
		image_name string)`

	//実行 結果は返ってこない為、_にする
    _, err := DbConnection.Exec(cmd)

    //エラーハンドリング
    if err != nil {
        c.Logger().Infof("エラー", err)
    }
	
	//Read
	cmd = "SELECT * FROM person"
    rows, _ := DbConnection.Query(cmd)
    defer rows.Close()
	
	// 取得したデータをループでスライスに追加　for rows.Next()
    var item []Item
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


	id := c.Param("item_id") 
	
	for i, ele := range item {
		if(strconv.Itoa(i) == id){
			return c.JSON(http.StatusOK, ele)
		}
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

	// Start server
	e.Logger.Fatal(e.Start(":9000"))
}
