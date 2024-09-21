package main

import (
    "log"
    "net/http"
    "app/user"
    "app/photo"
    "html/template"
)

func main() {
    // Get the PostgreSQL database connection
    postgresDB, err := user.GetPostgresDB()
    if err != nil {
        log.Fatalf("Failed to connect to PostgreSQL: %v", err)
    }
    defer postgresDB.Close()

    // Initialize the store with the PostgreSQL database connection
    userstore := user.NewUserStore(postgresDB)
    photostore:=photo.NewPhotostore(postgresDB)

    // Create a new Model instance
    userModel := user.NewModel(userstore)
    photoModel:=photo.NewModel(photostore)
 // Serve the registration form
 http.HandleFunc("/register/form", func(w http.ResponseWriter, r *http.Request) {
    t, err := template.ParseFiles("Registration.html")
    if err != nil {
        http.Error(w, "Failed to load registration page", http.StatusInternalServerError)
        return
    }
    t.Execute(w, nil)
})
  // Serve the login HTML form
  http.HandleFunc("/login/form", func(w http.ResponseWriter, r *http.Request) {
    t, _ := template.ParseFiles("Login.html")
    if err != nil {
        http.Error(w, "Failed to load registration page", http.StatusInternalServerError)
        return
    }
    t.Execute(w, nil)
})
http.HandleFunc("/photo", func(w http.ResponseWriter, r *http.Request) {
    t, _ := template.ParseFiles("photo.html")
    if err != nil {
        http.Error(w, "uploadimage", http.StatusInternalServerError)
        return
    }
    t.Execute(w, nil)
})
    // Define your routes
    http.HandleFunc("/", photo.HandleRoot)
    http.HandleFunc("/upload", photoModel.HandleUpload)
    http.HandleFunc("/serveimage/", photoModel.HandleServeImage)
    http.HandleFunc("/create_album", photoModel.HandleCreateAlbum)
    http.HandleFunc("/add_photo_to_album", photoModel.HandleAddPhotoToAlbum)
    http.HandleFunc("/get_photo_to_album",photoModel.HandleGetAlbumWithPhotos)

    // User-related routes
    http.HandleFunc("/register", userModel.RegisterUser)
    http.HandleFunc("/login", userModel.LoginUser)

    log.Println("Server is running on port 8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
