// Package docs Code generated by swaggo/swag. DO NOT EDIT
package docs

import "github.com/swaggo/swag"

const docTemplate = `{
    "schemes": {{ marshal .Schemes }},
    "swagger": "2.0",
    "info": {
        "description": "{{escape .Description}}",
        "title": "{{.Title}}",
        "contact": {},
        "version": "{{.Version}}"
    },
    "host": "{{.Host}}",
    "basePath": "{{.BasePath}}",
    "paths": {
        "/blog": {
            "post": {
                "description": "Create a new blog post with metadata and an optional image upload",
                "consumes": [
                    "multipart/form-data"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "blogs"
                ],
                "summary": "Create a new blog post",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Title",
                        "name": "title",
                        "in": "formData",
                        "required": true
                    },
                    {
                        "type": "string",
                        "description": "Meta Description",
                        "name": "meta_description",
                        "in": "formData"
                    },
                    {
                        "type": "string",
                        "description": "Focus Keyword",
                        "name": "focus_keyword",
                        "in": "formData"
                    },
                    {
                        "type": "string",
                        "description": "URL Keyword",
                        "name": "url_keyword",
                        "in": "formData",
                        "required": true
                    },
                    {
                        "type": "array",
                        "description": "Tags (comma-separated values or multiple fields)",
                        "name": "tags",
                        "in": "formData"
                    },
                    {
                        "type": "string",
                        "description": "Topic",
                        "name": "topic",
                        "in": "formData"
                    },
                    {
                        "type": "string",
                        "description": "Service",
                        "name": "service",
                        "in": "formData"
                    },
                    {
                        "type": "string",
                        "description": "Industry",
                        "name": "industry",
                        "in": "formData"
                    },
                    {
                        "type": "string",
                        "description": "Priority",
                        "name": "priority",
                        "in": "formData"
                    },
                    {
                        "type": "string",
                        "description": "Description",
                        "name": "description",
                        "in": "formData",
                        "required": true
                    },
                    {
                        "type": "file",
                        "description": "Image file (optional)",
                        "name": "image",
                        "in": "formData"
                    }
                ],
                "responses": {
                    "201": {
                        "description": "Created",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "type": "object",
                            "additionalProperties": {
                                "type": "string"
                            }
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "type": "object",
                            "additionalProperties": {
                                "type": "string"
                            }
                        }
                    }
                }
            }
        },
        "/blog/{urlKeyword}": {
            "get": {
                "description": "Retrieve a blog post by its URL keyword",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "blogs"
                ],
                "summary": "Get a blog post",
                "parameters": [
                    {
                        "type": "string",
                        "description": "URL Keyword of the blog post",
                        "name": "urlKeyword",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/main.BlogPost"
                        }
                    },
                    "404": {
                        "description": "Not Found",
                        "schema": {
                            "type": "object",
                            "additionalProperties": {
                                "type": "string"
                            }
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "type": "object",
                            "additionalProperties": {
                                "type": "string"
                            }
                        }
                    }
                }
            }
        },
        "/blogs": {
            "get": {
                "description": "Get a paginated list of blog posts",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "blogs"
                ],
                "summary": "List blog posts",
                "parameters": [
                    {
                        "type": "integer",
                        "description": "Page number",
                        "name": "page",
                        "in": "query"
                    },
                    {
                        "type": "integer",
                        "description": "Number of items per page",
                        "name": "pageSize",
                        "in": "query"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/main.PaginatedResponse"
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "type": "object",
                            "additionalProperties": {
                                "type": "string"
                            }
                        }
                    }
                }
            }
        },
        "/sitemap.xml": {
            "get": {
                "description": "Generate an XML sitemap of blog posts",
                "produces": [
                    "text/xml"
                ],
                "tags": [
                    "sitemap"
                ],
                "summary": "Generate sitemap.xml",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/main.Sitemap"
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "type": "object",
                            "additionalProperties": {
                                "type": "string"
                            }
                        }
                    }
                }
            }
        }
    },
    "definitions": {
        "main.BlogPost": {
            "type": "object",
            "properties": {
                "created_at": {
                    "type": "string"
                },
                "description": {
                    "type": "string"
                },
                "focus_keyword": {
                    "type": "string"
                },
                "id": {
                    "type": "integer"
                },
                "image": {
                    "type": "string"
                },
                "industry": {
                    "type": "string"
                },
                "meta_description": {
                    "type": "string"
                },
                "priority": {
                    "type": "string",
                    "enum": [
                        "maximum",
                        "high",
                        "normal"
                    ]
                },
                "service": {
                    "type": "string"
                },
                "tags": {
                    "type": "array",
                    "items": {
                        "type": "string"
                    }
                },
                "title": {
                    "type": "string"
                },
                "topic": {
                    "type": "string"
                },
                "updated_at": {
                    "type": "string"
                },
                "url_keyword": {
                    "type": "string"
                }
            }
        },
        "main.PaginatedResponse": {
            "type": "object",
            "properties": {
                "page": {
                    "description": "Current page number",
                    "type": "integer"
                },
                "pageSize": {
                    "description": "Number of items per page",
                    "type": "integer"
                },
                "posts": {
                    "description": "List of blog posts",
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/main.BlogPost"
                    }
                },
                "totalPages": {
                    "description": "Total number of pages",
                    "type": "integer"
                },
                "totalPosts": {
                    "description": "Total number of blog posts",
                    "type": "integer"
                }
            }
        },
        "main.Sitemap": {
            "type": "object",
            "properties": {
                "urls": {
                    "description": "List of URLs in the sitemap",
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/main.URL"
                    }
                }
            }
        },
        "main.URL": {
            "type": "object",
            "properties": {
                "changefreq": {
                    "description": "The change frequency of the URL",
                    "type": "string"
                },
                "loc": {
                    "description": "The URL of the blog post",
                    "type": "string"
                },
                "priority": {
                    "description": "The priority of the URL in the sitemap",
                    "type": "string"
                }
            }
        }
    }
}`

// SwaggerInfo holds exported Swagger Info so clients can modify it
var SwaggerInfo = &swag.Spec{
	Version:          "1.0",
	Host:             "",
	BasePath:         "/",
	Schemes:          []string{},
	Title:            "Blog API",
	Description:      "API for managing blog posts.",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  docTemplate,
	LeftDelim:        "{{",
	RightDelim:       "}}",
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
