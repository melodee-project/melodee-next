package handlers

import (
	"fmt"
	"melodee/internal/models"
	"melodee/open_subsonic/utils"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type PodcastHandler struct {
	DB *gorm.DB
}

func NewPodcastHandler(db *gorm.DB) *PodcastHandler {
	return &PodcastHandler{DB: db}
}

func (h *PodcastHandler) GetPodcasts(c *fiber.Ctx) error {
	includeEpisodes := c.Query("includeEpisodes") == "true"
	// id := c.Query("id") // Optional: filter by channel ID

	var channels []models.PodcastChannel
	query := h.DB.Model(&models.PodcastChannel{})

	if includeEpisodes {
		query = query.Preload("Episodes")
	}

	if err := query.Find(&channels).Error; err != nil {
		return utils.SendOpenSubsonicError(c, 70, "Could not fetch podcasts")
	}

	responseChannels := make([]utils.PodcastChannel, len(channels))
	for i, ch := range channels {
		responseChannels[i] = utils.PodcastChannel{
			ID:               fmt.Sprintf("%d", ch.ID),
			Url:              ch.URL,
			Title:            ch.Title,
			Description:      ch.Description,
			Status:           ch.Status,
			ErrorMessage:     ch.ErrorMessage,
			OriginalImageUrl: ch.ImageURL,
		}

		if includeEpisodes {
			episodes := make([]utils.PodcastEpisode, len(ch.Episodes))
			for j, ep := range ch.Episodes {
				episodes[j] = utils.PodcastEpisode{
					ID:          fmt.Sprintf("%d", ep.ID),
					StreamId:    fmt.Sprintf("%d", ep.ID), // Assuming stream ID is same as episode ID for now
					ChannelId:   fmt.Sprintf("%d", ch.ID),
					Title:       ep.Title,
					Description: ep.Description,
					PublishDate: utils.FormatTime(ep.PublishDate),
					Status:      ep.Status,
					Parent:      fmt.Sprintf("%d", ch.ID),
					IsDir:       false,
					Duration:    ep.Duration,
					Size:        ep.FileSize,
					ContentType: ep.ContentType,
					Suffix:      "mp3", // Default or derive from content type
					Path:        ep.FileName,
				}
			}
			responseChannels[i].Episodes = episodes
		}
	}

	return utils.SendResponse(c, &utils.OpenSubsonicResponse{
		Status:   "ok",
		Version:  "1.16.1",
		Podcasts: &utils.Podcasts{Channels: responseChannels},
	})
}

func (h *PodcastHandler) GetNewestPodcasts(c *fiber.Ctx) error {
	count := c.QueryInt("count", 20)
	if count > 50 {
		count = 50
	}

	var episodes []models.PodcastEpisode
	if err := h.DB.Order("publish_date desc").Limit(count).Preload("Channel").Find(&episodes).Error; err != nil {
		return utils.SendOpenSubsonicError(c, 70, "Could not fetch newest podcasts")
	}

	responseEpisodes := make([]utils.PodcastEpisode, len(episodes))
	for i, ep := range episodes {
		responseEpisodes[i] = utils.PodcastEpisode{
			ID:          fmt.Sprintf("%d", ep.ID),
			StreamId:    fmt.Sprintf("%d", ep.ID),
			ChannelId:   fmt.Sprintf("%d", ep.ChannelID),
			Title:       ep.Title,
			Description: ep.Description,
			PublishDate: utils.FormatTime(ep.PublishDate),
			Status:      ep.Status,
			Parent:      fmt.Sprintf("%d", ep.ChannelID),
			IsDir:       false,
			Duration:    ep.Duration,
			Size:        ep.FileSize,
			ContentType: ep.ContentType,
			Suffix:      "mp3",
			Path:        ep.FileName,
		}
		if ep.Channel != nil {
			responseEpisodes[i].CoverArt = ep.Channel.ImageURL // Use channel image as cover art
		}
	}

	return utils.SendResponse(c, &utils.OpenSubsonicResponse{
		Status:         "ok",
		Version:        "1.16.1",
		NewestPodcasts: &utils.NewestPodcasts{Episodes: responseEpisodes},
	})
}

func (h *PodcastHandler) RefreshPodcasts(c *fiber.Ctx) error {
	// In a real implementation, this would trigger a background job
	// For now, we just return success
	return utils.SendResponse(c, utils.SuccessResponse())
}

func (h *PodcastHandler) CreatePodcastChannel(c *fiber.Ctx) error {
	url := c.Query("url")
	if url == "" {
		return utils.SendOpenSubsonicError(c, 10, "Missing url parameter")
	}

	channel := models.PodcastChannel{
		URL:    url,
		Status: "new",
		Title:  "New Podcast", // Placeholder until fetched
	}

	if err := h.DB.Create(&channel).Error; err != nil {
		return utils.SendOpenSubsonicError(c, 0, "Could not create podcast channel")
	}

	// Trigger download/refresh logic here

	return utils.SendResponse(c, utils.SuccessResponse())
}

func (h *PodcastHandler) DeletePodcastChannel(c *fiber.Ctx) error {
	idStr := c.Query("id")
	if idStr == "" {
		return utils.SendOpenSubsonicError(c, 10, "Missing id parameter")
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return utils.SendOpenSubsonicError(c, 10, "Invalid id parameter")
	}

	if err := h.DB.Delete(&models.PodcastChannel{}, id).Error; err != nil {
		return utils.SendOpenSubsonicError(c, 70, "Podcast channel not found")
	}

	return utils.SendResponse(c, utils.SuccessResponse())
}

func (h *PodcastHandler) GetPodcastEpisode(c *fiber.Ctx) error {
	idStr := c.Query("id")
	if idStr == "" {
		return utils.SendOpenSubsonicError(c, 10, "Missing id parameter")
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return utils.SendOpenSubsonicError(c, 10, "Invalid id parameter")
	}

	var ep models.PodcastEpisode
	if err := h.DB.Preload("Channel").First(&ep, id).Error; err != nil {
		return utils.SendOpenSubsonicError(c, 70, "Podcast episode not found")
	}

	responseEpisode := utils.PodcastEpisode{
		ID:          fmt.Sprintf("%d", ep.ID),
		StreamId:    fmt.Sprintf("%d", ep.ID),
		ChannelId:   fmt.Sprintf("%d", ep.ChannelID),
		Title:       ep.Title,
		Description: ep.Description,
		PublishDate: utils.FormatTime(ep.PublishDate),
		Status:      ep.Status,
		Parent:      fmt.Sprintf("%d", ep.ChannelID),
		IsDir:       false,
		Duration:    ep.Duration,
		Size:        ep.FileSize,
		ContentType: ep.ContentType,
		Suffix:      "mp3",
		Path:        ep.FileName,
	}
	if ep.Channel != nil {
		responseEpisode.CoverArt = ep.Channel.ImageURL
	}

	return utils.SendResponse(c, &utils.OpenSubsonicResponse{
		Status:         "ok",
		Version:        "1.16.1",
		PodcastEpisode: &responseEpisode,
	})
}

func (h *PodcastHandler) DownloadPodcastEpisode(c *fiber.Ctx) error {
	idStr := c.Query("id")
	if idStr == "" {
		return utils.SendOpenSubsonicError(c, 10, "Missing id parameter")
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return utils.SendOpenSubsonicError(c, 10, "Invalid id parameter")
	}

	var ep models.PodcastEpisode
	if err := h.DB.First(&ep, id).Error; err != nil {
		return utils.SendOpenSubsonicError(c, 70, "Podcast episode not found")
	}

	ep.Status = "downloading"
	h.DB.Save(&ep)

	return utils.SendResponse(c, utils.SuccessResponse())
}

func (h *PodcastHandler) DeletePodcastEpisode(c *fiber.Ctx) error {
	idStr := c.Query("id")
	if idStr == "" {
		return utils.SendOpenSubsonicError(c, 10, "Missing id parameter")
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return utils.SendOpenSubsonicError(c, 10, "Invalid id parameter")
	}

	if err := h.DB.Delete(&models.PodcastEpisode{}, id).Error; err != nil {
		return utils.SendOpenSubsonicError(c, 70, "Podcast episode not found")
	}

	return utils.SendResponse(c, utils.SuccessResponse())
}
