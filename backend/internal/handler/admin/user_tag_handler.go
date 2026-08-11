package admin

import (
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type UserTagHandler struct {
	tagService *service.UserTagService
}

func NewUserTagHandler(tagService *service.UserTagService) *UserTagHandler {
	return &UserTagHandler{tagService: tagService}
}

type userTagResponse struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type userTagIDsRequest struct {
	TagIDs []int64 `json:"tag_ids"`
}

type userIDsRequest struct {
	UserIDs []int64 `json:"user_ids"`
}

type tagUserResponse struct {
	ID       int64  `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Status   string `json:"status"`
}

func userTagToResponse(tag *service.UserTag) *userTagResponse {
	if tag == nil {
		return nil
	}
	return &userTagResponse{
		ID:        tag.ID,
		Name:      tag.Name,
		CreatedAt: tag.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: tag.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func userTagsToResponse(tags []service.UserTag) []*userTagResponse {
	result := make([]*userTagResponse, 0, len(tags))
	for i := range tags {
		result = append(result, userTagToResponse(&tags[i]))
	}
	return result
}

func (h *UserTagHandler) List(c *gin.Context) {
	tags, err := h.tagService.List(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, userTagsToResponse(tags))
}

type createUserTagRequest struct {
	Name string `json:"name" binding:"required"`
}

func (h *UserTagHandler) Create(c *gin.Context) {
	var req createUserTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	tag, err := h.tagService.Create(c.Request.Context(), service.CreateUserTagInput{Name: req.Name})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, userTagToResponse(tag))
}

type updateUserTagRequest struct {
	Name string `json:"name" binding:"required"`
}

func (h *UserTagHandler) Update(c *gin.Context) {
	id, err := parseUserTagID(c)
	if err != nil {
		response.BadRequest(c, "Invalid tag ID")
		return
	}

	var req updateUserTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	tag, err := h.tagService.Update(c.Request.Context(), id, service.UpdateUserTagInput{Name: req.Name})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, userTagToResponse(tag))
}

func (h *UserTagHandler) Delete(c *gin.Context) {
	id, err := parseUserTagID(c)
	if err != nil {
		response.BadRequest(c, "Invalid tag ID")
		return
	}

	if err := h.tagService.Delete(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "User tag deleted successfully"})
}

func (h *UserTagHandler) GetTagUsers(c *gin.Context) {
	tagID, err := parseUserTagID(c)
	if err != nil {
		response.BadRequest(c, "Invalid tag ID")
		return
	}
	page, pageSize := response.ParsePagination(c)
	search := strings.TrimSpace(c.Query("search"))
	if runes := []rune(search); len(runes) > 100 {
		search = string(runes[:100])
	}
	status := strings.TrimSpace(c.Query("status"))
	if status != "" && status != service.StatusActive && status != service.StatusDisabled {
		response.BadRequest(c, "Invalid user status")
		return
	}

	users, total, err := h.tagService.ListTagUsers(c.Request.Context(), tagID, page, pageSize, search, status)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	items := make([]tagUserResponse, 0, len(users))
	for i := range users {
		items = append(items, tagUserResponse{
			ID:       users[i].ID,
			Email:    users[i].Email,
			Username: users[i].Username,
			Status:   users[i].Status,
		})
	}
	response.Paginated(c, items, total, page, pageSize)
}

func (h *UserTagHandler) AddUsersToTag(c *gin.Context) {
	tagID, err := parseUserTagID(c)
	if err != nil {
		response.BadRequest(c, "Invalid tag ID")
		return
	}
	var req userIDsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	affected, err := h.tagService.AddUsersToTag(c.Request.Context(), tagID, req.UserIDs)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"affected": affected})
}

func (h *UserTagHandler) GetUserTags(c *gin.Context) {
	userID, err := parseUserTagID(c)
	if err != nil {
		response.BadRequest(c, "Invalid user ID")
		return
	}

	tags, err := h.tagService.GetUserTags(c.Request.Context(), userID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, userTagsToResponse(tags))
}

func (h *UserTagHandler) UpdateUserTags(c *gin.Context) {
	userID, err := parseUserTagID(c)
	if err != nil {
		response.BadRequest(c, "Invalid user ID")
		return
	}

	var req userTagIDsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	tags, err := h.tagService.ReplaceUserTags(c.Request.Context(), userID, req.TagIDs)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, userTagsToResponse(tags))
}

func (h *UserTagHandler) GetGroupTags(c *gin.Context) {
	groupID, err := parseUserTagID(c)
	if err != nil {
		response.BadRequest(c, "Invalid group ID")
		return
	}

	tags, err := h.tagService.GetGroupTags(c.Request.Context(), groupID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, userTagsToResponse(tags))
}

func (h *UserTagHandler) UpdateGroupTags(c *gin.Context) {
	groupID, err := parseUserTagID(c)
	if err != nil {
		response.BadRequest(c, "Invalid group ID")
		return
	}

	var req userTagIDsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	tags, err := h.tagService.ReplaceGroupTags(c.Request.Context(), groupID, req.TagIDs)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, userTagsToResponse(tags))
}

func parseUserTagID(c *gin.Context) (int64, error) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, strconv.ErrSyntax
	}
	return id, nil
}
