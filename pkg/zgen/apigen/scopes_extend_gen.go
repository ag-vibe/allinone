package apigen

import (
	"github.com/gofiber/fiber/v2"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

type Validator interface { 
    // AuthFunc is called before the request is processed. The response will be 401 if the auth fails.
    AuthFunc(*fiber.Ctx) error

    // PreValidate is called before the request is processed. The response will be 403 if the validation fails.
    PreValidate(*fiber.Ctx) error
    
    // PostValidate is called after the request is processed. The response will be 403 if the validation fails.
    PostValidate(*fiber.Ctx) error

    OperationPermit(c *fiber.Ctx, operationID string) error
 }


type XMiddleware struct {
	ServerInterface
	Validator
}

func NewXMiddleware(handler ServerInterface, validator Validator) ServerInterface {
	return &XMiddleware{ServerInterface: handler, Validator: validator}
}

// Upload attachment
// (POST /attachments)
func (x *XMiddleware) UploadAttachment(c *fiber.Ctx) error {
    if err := x.AuthFunc(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString(err.Error())
	} 
	if err := x.PreValidate(c); err != nil {
		return c.Status(fiber.StatusForbidden).SendString(err.Error())
	}
	   
	if err := x.PostValidate(c); err != nil {
		return c.Status(fiber.StatusForbidden).SendString(err.Error())
	}
    return x.ServerInterface.UploadAttachment(c)
}
// Delete attachment
// (DELETE /attachments/{id})
func (x *XMiddleware) DeleteAttachment(c *fiber.Ctx, id openapi_types.UUID) error {
    if err := x.AuthFunc(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString(err.Error())
	} 
	if err := x.PreValidate(c); err != nil {
		return c.Status(fiber.StatusForbidden).SendString(err.Error())
	}
	   
	if err := x.PostValidate(c); err != nil {
		return c.Status(fiber.StatusForbidden).SendString(err.Error())
	}
    return x.ServerInterface.DeleteAttachment(c, id)
}
// Get attachment metadata
// (GET /attachments/{id})
func (x *XMiddleware) GetAttachment(c *fiber.Ctx, id openapi_types.UUID) error {
    if err := x.AuthFunc(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString(err.Error())
	} 
	if err := x.PreValidate(c); err != nil {
		return c.Status(fiber.StatusForbidden).SendString(err.Error())
	}
	   
	if err := x.PostValidate(c); err != nil {
		return c.Status(fiber.StatusForbidden).SendString(err.Error())
	}
    return x.ServerInterface.GetAttachment(c, id)
}
// Download attachment content
// (GET /attachments/{id}/content)
func (x *XMiddleware) DownloadAttachment(c *fiber.Ctx, id openapi_types.UUID) error {
    if err := x.AuthFunc(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString(err.Error())
	} 
	if err := x.PreValidate(c); err != nil {
		return c.Status(fiber.StatusForbidden).SendString(err.Error())
	}
	   
	if err := x.PostValidate(c); err != nil {
		return c.Status(fiber.StatusForbidden).SendString(err.Error())
	}
    return x.ServerInterface.DownloadAttachment(c, id)
}
// Unlink attachment from a resource
// (DELETE /attachments/{id}/links)
func (x *XMiddleware) UnlinkAttachment(c *fiber.Ctx, id openapi_types.UUID) error {
    if err := x.AuthFunc(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString(err.Error())
	} 
	if err := x.PreValidate(c); err != nil {
		return c.Status(fiber.StatusForbidden).SendString(err.Error())
	}
	   
	if err := x.PostValidate(c); err != nil {
		return c.Status(fiber.StatusForbidden).SendString(err.Error())
	}
    return x.ServerInterface.UnlinkAttachment(c, id)
}
// Link attachment to a resource
// (POST /attachments/{id}/links)
func (x *XMiddleware) LinkAttachment(c *fiber.Ctx, id openapi_types.UUID) error {
    if err := x.AuthFunc(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString(err.Error())
	} 
	if err := x.PreValidate(c); err != nil {
		return c.Status(fiber.StatusForbidden).SendString(err.Error())
	}
	   
	if err := x.PostValidate(c); err != nil {
		return c.Status(fiber.StatusForbidden).SendString(err.Error())
	}
    return x.ServerInterface.LinkAttachment(c, id)
}
// Increment Counter
// (POST /counter)
func (x *XMiddleware) IncrementCounter(c *fiber.Ctx) error {
    if err := x.AuthFunc(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString(err.Error())
	} 
	if err := x.PreValidate(c); err != nil {
		return c.Status(fiber.StatusForbidden).SendString(err.Error())
	}
	operationID := "IncrementCounter"  
	if err := x.OperationPermit(c, operationID); err != nil {
	    return c.Status(fiber.StatusForbidden).SendString(err.Error())
	}  
	if err := x.PostValidate(c); err != nil {
		return c.Status(fiber.StatusForbidden).SendString(err.Error())
	}
    return x.ServerInterface.IncrementCounter(c)
}
// List attachments by resource
// (GET /resources/{resourceType}/{resourceId}/attachments)
func (x *XMiddleware) ListAttachmentsByResource(c *fiber.Ctx, resourceType string, resourceId openapi_types.UUID) error {
    if err := x.AuthFunc(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString(err.Error())
	} 
	if err := x.PreValidate(c); err != nil {
		return c.Status(fiber.StatusForbidden).SendString(err.Error())
	}
	   
	if err := x.PostValidate(c); err != nil {
		return c.Status(fiber.StatusForbidden).SendString(err.Error())
	}
    return x.ServerInterface.ListAttachmentsByResource(c, resourceType, resourceId)
}
// List TODO items for current user
// (GET /todos)
func (x *XMiddleware) ListTodos(c *fiber.Ctx) error {
    if err := x.AuthFunc(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString(err.Error())
	} 
	if err := x.PreValidate(c); err != nil {
		return c.Status(fiber.StatusForbidden).SendString(err.Error())
	}
	   
	if err := x.PostValidate(c); err != nil {
		return c.Status(fiber.StatusForbidden).SendString(err.Error())
	}
    return x.ServerInterface.ListTodos(c)
}
// Create a new TODO item
// (POST /todos)
func (x *XMiddleware) CreateTodo(c *fiber.Ctx) error {
    if err := x.AuthFunc(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString(err.Error())
	} 
	if err := x.PreValidate(c); err != nil {
		return c.Status(fiber.StatusForbidden).SendString(err.Error())
	}
	   
	if err := x.PostValidate(c); err != nil {
		return c.Status(fiber.StatusForbidden).SendString(err.Error())
	}
    return x.ServerInterface.CreateTodo(c)
}
// Delete a TODO item
// (DELETE /todos/{id})
func (x *XMiddleware) DeleteTodo(c *fiber.Ctx, id openapi_types.UUID) error {
    if err := x.AuthFunc(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString(err.Error())
	} 
	if err := x.PreValidate(c); err != nil {
		return c.Status(fiber.StatusForbidden).SendString(err.Error())
	}
	   
	if err := x.PostValidate(c); err != nil {
		return c.Status(fiber.StatusForbidden).SendString(err.Error())
	}
    return x.ServerInterface.DeleteTodo(c, id)
}
// Update a TODO item
// (PATCH /todos/{id})
func (x *XMiddleware) UpdateTodo(c *fiber.Ctx, id openapi_types.UUID) error {
    if err := x.AuthFunc(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString(err.Error())
	} 
	if err := x.PreValidate(c); err != nil {
		return c.Status(fiber.StatusForbidden).SendString(err.Error())
	}
	   
	if err := x.PostValidate(c); err != nil {
		return c.Status(fiber.StatusForbidden).SendString(err.Error())
	}
    return x.ServerInterface.UpdateTodo(c, id)
}

