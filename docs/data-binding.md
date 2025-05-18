# Data Binding and Validation

MoraRouter provides powerful data binding and validation capabilities that make it easy to handle incoming request data safely and efficiently.

## What is Data Binding?

Data binding is the process of converting incoming request data (JSON, XML, form data, etc.) into Go structs. This makes the data easy to work with in a type-safe manner while validating that it meets your requirements.

## JSON Binding

The most common use case is binding JSON request bodies:

```go
type CreateUserRequest struct {
    Name     string `json:"name" validate:"required,min=3"`
    Email    string `json:"email" validate:"required,email"`
    Age      int    `json:"age" validate:"min=18"`
    Password string `json:"password" validate:"required,min=8"`
}

// Manual approach
r.Post("/users", func(w http.ResponseWriter, r *http.Request, p router.Params) {
    var req CreateUserRequest
    
    // Parse JSON body
    if err := router.ParseJSON(r, &req); err != nil {
        router.Error(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
        return
    }
    
    // Validate the request
    if err := router.Validate(req); err != nil {
        router.Error(w, http.StatusBadRequest, "Validation failed: "+err.Error())
        return
    }
    
    // Now use the validated request
    // ...
    
    router.JSON(w, http.StatusCreated, map[string]interface{}{
        "id": "123",
        "name": req.Name,
        "email": req.Email,
    })
})
```

### Using BindJSON Helper

MoraRouter provides a convenient `BindJSON` helper that combines parsing and validation:

```go
r.Post("/users", router.BindJSON(func(w http.ResponseWriter, r *http.Request, p router.Params, req CreateUserRequest) {
    // req is already parsed and validated!
    // If validation failed, an error response is already sent
    
    // Use the validated request
    user := createUser(req)
    
    router.JSON(w, http.StatusCreated, user)
}))
```

## XML Binding

MoraRouter also supports XML binding:

```go
type Product struct {
    SKU         string  `xml:"sku" validate:"required"`
    Name        string  `xml:"name" validate:"required"`
    Price       float64 `xml:"price" validate:"gt=0"`
    Description string  `xml:"description"`
}

r.Post("/products", router.BindXML(func(w http.ResponseWriter, r *http.Request, p router.Params, req Product) {
    // req is parsed from XML and validated
    
    // Use the validated request
    // ...
    
    router.XML(w, http.StatusCreated, req)
}))
```

## Form Data Binding

For handling HTML form submissions:

```go
type ContactForm struct {
    Name    string `form:"name" validate:"required"`
    Email   string `form:"email" validate:"required,email"`
    Message string `form:"message" validate:"required,min=10"`
}

r.Post("/contact", router.BindForm(func(w http.ResponseWriter, r *http.Request, p router.Params, form router.Form, req ContactForm) {
    // req is populated from form fields and validated
    
    // Use the validated form data
    sendEmail(req.Email, req.Name, req.Message)
    
    // Check for form errors
    if form.HasErrors() {
        // Handle form errors
        router.JSON(w, http.StatusBadRequest, form.GetErrors())
        return
    }
    
    // Redirect after successful submission
    router.Redirect(w, r, "/contact/thank-you", http.StatusSeeOther)
}))
```

## File Uploads

MoraRouter makes file uploads easy with the Form object:

```go
type ProfileForm struct {
    Name   string           `form:"name" validate:"required"`
    Avatar *router.FormFile `form:"avatar" validate:"required"`
    Bio    string           `form:"bio"`
}

r.Post("/profile", router.BindForm(func(w http.ResponseWriter, r *http.Request, p router.Params, form router.Form, req ProfileForm) {
    // Check for file
    if req.Avatar == nil {
        router.Error(w, http.StatusBadRequest, "Avatar is required")
        return
    }
    
    // Save the uploaded file
    filePath, err := form.SaveFile("avatar", "./uploads")
    if err != nil {
        router.Error(w, http.StatusInternalServerError, "Failed to save file")
        return
    }
    
    // Use the file path and other form data
    updateProfile(p["userId"], req.Name, filePath, req.Bio)
    
    router.JSON(w, http.StatusOK, map[string]interface{}{
        "message": "Profile updated",
        "avatar_path": filePath,
    })
}))
```

### Multiple File Uploads

Handling multiple files:

```go
type GalleryForm struct {
    Title  string             `form:"title" validate:"required"`
    Photos []*router.FormFile `form:"photos" validate:"required,min=1"`
}

r.Post("/gallery", router.BindForm(func(w http.ResponseWriter, r *http.Request, p router.Params, form router.Form, req GalleryForm) {
    // Check for files
    if len(req.Photos) == 0 {
        router.Error(w, http.StatusBadRequest, "At least one photo is required")
        return
    }
    
    // Save all uploaded files
    filePaths := []string{}
    for i, photo := range req.Photos {
        // Generate unique filename
        filename := fmt.Sprintf("%d-%s", time.Now().Unix(), photo.Filename)
        
        // Save the file
        path, err := form.SaveFileWithName("photos", i, "./uploads/gallery", filename)
        if err != nil {
            router.Error(w, http.StatusInternalServerError, "Failed to save file")
            return
        }
        
        filePaths = append(filePaths, path)
    }
    
    // Use the file paths and other form data
    createGallery(req.Title, filePaths)
    
    router.JSON(w, http.StatusOK, map[string]interface{}{
        "message": "Gallery created",
        "photo_count": len(filePaths),
    })
}))
```

## URL Parameter Binding

You can also bind URL parameters to a struct:

```go
type PostParams struct {
    Year  int    `param:"year" validate:"required,min=2000,max=2030"`
    Month int    `param:"month" validate:"required,min=1,max=12"`
    Slug  string `param:"slug" validate:"required"`
}

r.Get("/posts/:year/:month/:slug", router.BindParams(func(w http.ResponseWriter, r *http.Request, p router.Params, params PostParams) {
    // params is populated from URL parameters and validated
    
    // Use the validated parameters
    post := getPost(params.Year, params.Month, params.Slug)
    
    router.JSON(w, http.StatusOK, post)
}))
```

## Query Parameter Binding

Similarly, you can bind query parameters:

```go
type SearchQuery struct {
    Query  string `query:"q" validate:"required,min=3"`
    Page   int    `query:"page" validate:"min=1"`
    Limit  int    `query:"limit" validate:"min=1,max=100"`
    SortBy string `query:"sort" validate:"oneof=date title popularity"`
}

r.Get("/search", router.BindQuery(func(w http.ResponseWriter, r *http.Request, p router.Params, q SearchQuery) {
    // q is populated from query parameters and validated
    // Default values are applied if parameters are missing
    
    // Use the validated query parameters
    results := search(q.Query, q.Page, q.Limit, q.SortBy)
    
    router.JSON(w, http.StatusOK, results)
}))
```

## Combined Binding

You can combine multiple binding sources:

```go
type CreatePostRequest struct {
    Title   string `json:"title" validate:"required"`
    Content string `json:"content" validate:"required"`
}

type PostContext struct {
    UserID string `param:"user_id" validate:"required"`
}

r.Post("/users/:user_id/posts", func(w http.ResponseWriter, r *http.Request, p router.Params) {
    // Bind URL parameters
    var ctx PostContext
    if err := router.BindParamsStruct(p, &ctx); err != nil {
        router.Error(w, http.StatusBadRequest, err.Error())
        return
    }
    
    // Bind JSON body
    var req CreatePostRequest
    if err := router.ParseJSON(r, &req); err != nil {
        router.Error(w, http.StatusBadRequest, err.Error())
        return
    }
    
    // Validate request
    if err := router.Validate(req); err != nil {
        router.Error(w, http.StatusBadRequest, err.Error())
        return
    }
    
    // Use both bound objects
    post := createPost(ctx.UserID, req.Title, req.Content)
    
    router.JSON(w, http.StatusCreated, post)
})
```

## Validation Rules

MoraRouter supports many validation rules through the `validate` tag:

| Rule | Description | Example |
|------|-------------|---------|
| `required` | Field cannot be empty | `validate:"required"` |
| `min=n` | Minimum length for strings, minimum value for numbers | `validate:"min=3"` |
| `max=n` | Maximum length for strings, maximum value for numbers | `validate:"max=100"` |
| `email` | Must be a valid email address | `validate:"email"` |
| `url` | Must be a valid URL | `validate:"url"` |
| `oneof=a b c` | Must be one of the provided values | `validate:"oneof=active pending inactive"` |
| `gt=n` | Greater than (for numbers) | `validate:"gt=0"` |
| `lt=n` | Less than (for numbers) | `validate:"lt=1000"` |
| `gte=n` | Greater than or equal (for numbers) | `validate:"gte=18"` |
| `lte=n` | Less than or equal (for numbers) | `validate:"lte=65"` |
| `uuid` | Must be a valid UUID | `validate:"uuid"` |
| `alpha` | Must contain only letters | `validate:"alpha"` |
| `alphanum` | Must contain only letters and numbers | `validate:"alphanum"` |
| `numeric` | Must contain only numbers | `validate:"numeric"` |
| `len=n` | Exact length for strings, exact size for slices/maps | `validate:"len=10"` |
| `regexp=pattern` | Must match the regular expression | `validate:"regexp=^[A-Z][a-z]+$"` |

You can combine multiple rules:

```go
type User struct {
    Username string `json:"username" validate:"required,alphanum,min=4,max=20"`
    Email    string `json:"email" validate:"required,email"`
    Age      int    `json:"age" validate:"required,gte=18,lte=120"`
    Role     string `json:"role" validate:"required,oneof=user admin moderator"`
}
```

## Custom Validation

You can create custom validators for complex rules:

```go
// Register a custom validator
router.RegisterValidator("strong_password", func(value interface{}) bool {
    password, ok := value.(string)
    if !ok {
        return false
    }
    
    // Password must have at least one uppercase, lowercase, digit, and special char
    hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
    hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
    hasDigit := regexp.MustCompile(`[0-9]`).MatchString(password)
    hasSpecial := regexp.MustCompile(`[^a-zA-Z0-9]`).MatchString(password)
    
    return hasUpper && hasLower && hasDigit && hasSpecial
})

// Use the custom validator
type RegisterRequest struct {
    Username string `json:"username" validate:"required,alphanum,min=4,max=20"`
    Password string `json:"password" validate:"required,min=8,strong_password"`
    Email    string `json:"email" validate:"required,email"`
}
```

## Handling Validation Errors

By default, the binding helpers will send a 400 Bad Request response with validation errors. You can customize this behavior:

```go
r.Post("/users", func(w http.ResponseWriter, r *http.Request, p router.Params) {
    var req CreateUserRequest
    
    // Parse JSON
    if err := router.ParseJSON(r, &req); err != nil {
        router.JSON(w, http.StatusBadRequest, map[string]interface{}{
            "status": "error",
            "message": "Invalid JSON format",
            "error": err.Error(),
        })
        return
    }
    
    // Validate
    if errs := router.ValidateDetailed(req); len(errs) > 0 {
        // Return detailed validation errors
        router.JSON(w, http.StatusBadRequest, map[string]interface{}{
            "status": "error",
            "message": "Validation failed",
            "errors": errs,
        })
        return
    }
    
    // Process valid request...
})
```

## Default Values

You can provide default values for fields using the `default` tag:

```go
type ListOptions struct {
    Page  int    `query:"page" default:"1" validate:"min=1"`
    Limit int    `query:"limit" default:"20" validate:"min=1,max=100"`
    Sort  string `query:"sort" default:"created_at"`
}

r.Get("/posts", router.BindQuery(func(w http.ResponseWriter, r *http.Request, p router.Params, opts ListOptions) {
    // If ?page= is not provided, opts.Page will be 1
    // If ?limit= is not provided, opts.Limit will be 20
    // If ?sort= is not provided, opts.Sort will be "created_at"
    
    posts := getPosts(opts.Page, opts.Limit, opts.Sort)
    
    router.JSON(w, http.StatusOK, posts)
}))
```

## Nested Objects and Slices

You can bind complex nested structures:

```go
type Address struct {
    Street  string `json:"street" validate:"required"`
    City    string `json:"city" validate:"required"`
    State   string `json:"state" validate:"required,len=2"`
    ZipCode string `json:"zip_code" validate:"required,numeric,len=5"`
}

type Phone struct {
    Type   string `json:"type" validate:"required,oneof=home work mobile"`
    Number string `json:"number" validate:"required,numeric,len=10"`
}

type CreateUserRequest struct {
    Name        string    `json:"name" validate:"required"`
    Email       string    `json:"email" validate:"required,email"`
    Password    string    `json:"password" validate:"required,min=8"`
    Address     Address   `json:"address" validate:"required"`
    PhoneNumbers []Phone  `json:"phone_numbers" validate:"dive"`
}

r.Post("/users", router.BindJSON(func(w http.ResponseWriter, r *http.Request, p router.Params, req CreateUserRequest) {
    // All nested validations are performed
    
    // Use the validated request
    // ...
    
    router.JSON(w, http.StatusCreated, map[string]string{
        "id": "123",
        "message": "User created",
    })
}))
```

## Conclusion

MoraRouter's data binding and validation features make it easy to safely handle incoming requests. By using these tools, you can:

- Reduce boilerplate code for parsing request data
- Validate user input with clear rules
- Provide helpful error messages
- Handle complex data structures

Next, check out [Responses](responses.md) to learn about different ways to format your API responses.
