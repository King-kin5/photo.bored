package photo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Photo struct {
	photo_id uuid.UUID
	Filename string
	Data     []byte
	Date     time.Time
	MIMEType string
	Location string
}

type Album struct {
	ID        uuid.UUID
	Name      string
	CreatedAt time.Time
	Photos    []Photo
}

func (m *Model) HandleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		log.Printf("Invalid method for upload: %v", r.Method)
		return
	}

	log.Printf("Content-Type of the request: %v", r.Header.Get("Content-Type"))

	err := r.ParseMultipartForm(10 << 20) // 10 MB max
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Printf("Error parsing multipart form: %v", err)
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Printf("Error retrieving file from form: %v", err)
		return
	}
	defer file.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, file); err != nil {
		http.Error(w, "Failed to read uploaded file", http.StatusInternalServerError)
		log.Printf("Failed to read the uploaded file: %v", err)
		return
	}

	mimeType := http.DetectContentType(buf.Bytes())

	image := &Photo{
		photo_id: uuid.New(),
		Filename: header.Filename,
		Data:     buf.Bytes(),
		Date:     time.Now(),
		Location: "Unknown Location",
		MIMEType: mimeType,
	}

	if err := m.store.Addimage(image); err != nil {
		http.Error(w, "Failed to save image", http.StatusInternalServerError)
		log.Printf("Failed to save image to the database: %v", err)
		return
	}

	imageURL := fmt.Sprintf("/serveimage/%s", image.Filename)
	json.NewEncoder(w).Encode(map[string]string{"imageURL": imageURL})
	fmt.Fprintf(w, "Photo uploaded successfully with ID: %s", image.photo_id)
}

func HandleRoot(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "index.html")
}

func (m *Model) HandleServeImage(w http.ResponseWriter, r *http.Request) {
	// Ensure the method is GET
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		log.Printf("Invalid method for serving image: %v", r.Method)
		return
	}

	// Parse the URL to get the filename
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) != 3 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		log.Printf("Invalid URL path: %v", r.URL.Path)
		return
	}
	filename := parts[2]

	// Retrieve the image from the database
	photo, err := m.store.GetImageByFilename(filename)
	if err != nil {
		http.Error(w, "Image not found", http.StatusNotFound)
		log.Printf("Image not found: %v", filename)
		return
	}

	// Check if the photo object is nil
	if photo == nil {
		http.Error(w, "Image not found", http.StatusNotFound)
		log.Printf("Image not found: %v", filename)
		return
	}
	// Set the Content-Disposition header to serve the image with the original filename
	w.Header().Set("Content-Disposition", "inline; filename="+photo.Filename)
	// Set the Content-Type header based on the image's MIME type
	w.Header().Set("Content-Type", photo.MIMEType)
	// Serve the image
	_, err = w.Write(photo.Data)
	if err != nil {
		http.Error(w, "Failed to serve image", http.StatusInternalServerError)
		log.Printf("Failed to serve image %v: %v", filename, err)
	}
}

// HandelDelete
func (m *Model) HandelDeleteimage(w http.ResponseWriter, r *http.Request) {
	//EnURE THE METHOD IS DELETE
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowwed", http.StatusMethodNotAllowed)
	}
	//Parse the URL to get the filename
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) != 3 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	filename := parts[2]
	photo, err := m.store.DeleteImageByFilename(filename)
	if err != nil {
		http.Error(w, "Could not delete iamge", http.StatusInternalServerError)
		return
	}
	// Ensure that the photo is not nil
	if photo == nil {
		http.Error(w, "Image not found", http.StatusNotFound)
		return
	}
	// Set the Content-Disposition header to serve the image with the original filename
	w.Header().Set("Content-Disposition", "inline; filename="+photo.Filename)
	fmt.Fprint(w, "Image Deleted successfully", photo.Filename)
}

// HanldeCaption
func (m *Model) Handelcaption(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowwed", http.StatusMethodNotAllowed)
	}
	//Parse the URL to get the filename
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) != 3 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	filename := parts[2]
	// Parse the form to get the new caption
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}
	newCaption := r.FormValue("caption")
	if newCaption == "" {
		http.Error(w, "Caption cannot be empty", http.StatusBadRequest)
		return
	}
	// Update the caption in the database
	err = m.store.UpdateCaptionByFilename(filename, newCaption)
	if err != nil {
		http.Error(w, "Failed to update caption", http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "Caption updated successfully")
}

func (m *Model) HandleCreateAlbum(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	name := r.FormValue("Name")
	if name == "" {
		http.Error(w, "Album name is required", http.StatusBadRequest)
		return
	}

	album := &Album{
		ID:        uuid.New(),
		Name:      name,
		CreatedAt: time.Now(),
	}

	err := m.store.Createalbum(album)
	if err != nil {
		http.Error(w, "Failed to create album", http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "Album created successfully with ID: %s", album.ID)
}

// Add Existing Photo to Album Function
func (m *Model) HandleAddPhotoToAlbum(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	albumIDStr := r.FormValue("AlbumID")
	photoIDStr := r.FormValue("PhotoID")

	if albumIDStr == "" || photoIDStr == "" {
		http.Error(w, "Both AlbumID and PhotoID are required", http.StatusBadRequest)
		return
	}
	albumID, err := uuid.Parse(albumIDStr)
	if err != nil {
		http.Error(w, "Invalid AlbumID", http.StatusBadRequest)
		return
	}
	photoID, err := uuid.Parse(photoIDStr)
	if err != nil {
		http.Error(w, "Invalid PhotoID", http.StatusBadRequest)
		return
	}
	err = m.store.AddPhotoToAlbum(albumID, photoID)
	if err != nil {
		http.Error(w, "Failed to add photo to album", http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "Photo added to album successfully")
}
