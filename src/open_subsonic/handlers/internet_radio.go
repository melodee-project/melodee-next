package handlers

import (
	"fmt"
	"melodee/internal/models"
	"melodee/open_subsonic/utils"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type InternetRadioHandler struct {
	DB *gorm.DB
}

func NewInternetRadioHandler(db *gorm.DB) *InternetRadioHandler {
	return &InternetRadioHandler{DB: db}
}

func (h *InternetRadioHandler) GetInternetRadioStations(c *fiber.Ctx) error {
	var stations []models.RadioStation
	if err := h.DB.Find(&stations).Error; err != nil {
		return utils.SendOpenSubsonicError(c, 70, "Could not fetch internet radio stations")
	}

	responseStations := make([]utils.InternetRadioStation, len(stations))
	for i, st := range stations {
		responseStations[i] = utils.InternetRadioStation{
			ID:          fmt.Sprintf("%d", st.ID),
			Name:        st.Name,
			StreamUrl:   st.StreamURL,
			HomePageUrl: st.HomePageURL,
		}
	}

	return utils.SendResponse(c, &utils.OpenSubsonicResponse{
		Status:                "ok",
		Version:               "1.16.1",
		InternetRadioStations: &utils.InternetRadioStations{Stations: responseStations},
	})
}

func (h *InternetRadioHandler) CreateInternetRadioStation(c *fiber.Ctx) error {
	streamUrl := c.Query("streamUrl")
	name := c.Query("name")
	homepageUrl := c.Query("homepageUrl")

	if streamUrl == "" || name == "" {
		return utils.SendOpenSubsonicError(c, 10, "Missing required parameters")
	}

	station := models.RadioStation{
		StreamURL:   streamUrl,
		Name:        name,
		HomePageURL: homepageUrl,
		IsEnabled:   true,
	}

	if err := h.DB.Create(&station).Error; err != nil {
		return utils.SendOpenSubsonicError(c, 0, "Could not create internet radio station")
	}

	return utils.SendResponse(c, utils.SuccessResponse())
}

func (h *InternetRadioHandler) UpdateInternetRadioStation(c *fiber.Ctx) error {
	idStr := c.Query("id")
	if idStr == "" {
		return utils.SendOpenSubsonicError(c, 10, "Missing id parameter")
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return utils.SendOpenSubsonicError(c, 10, "Invalid id parameter")
	}

	var station models.RadioStation
	if err := h.DB.First(&station, id).Error; err != nil {
		return utils.SendOpenSubsonicError(c, 70, "Internet radio station not found")
	}

	streamUrl := c.Query("streamUrl")
	name := c.Query("name")
	homepageUrl := c.Query("homepageUrl")

	if streamUrl != "" {
		station.StreamURL = streamUrl
	}
	if name != "" {
		station.Name = name
	}
	if homepageUrl != "" {
		station.HomePageURL = homepageUrl
	}

	if err := h.DB.Save(&station).Error; err != nil {
		return utils.SendOpenSubsonicError(c, 0, "Could not update internet radio station")
	}

	return utils.SendResponse(c, utils.SuccessResponse())
}

func (h *InternetRadioHandler) DeleteInternetRadioStation(c *fiber.Ctx) error {
	idStr := c.Query("id")
	if idStr == "" {
		return utils.SendOpenSubsonicError(c, 10, "Missing id parameter")
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return utils.SendOpenSubsonicError(c, 10, "Invalid id parameter")
	}

	if err := h.DB.Delete(&models.RadioStation{}, id).Error; err != nil {
		return utils.SendOpenSubsonicError(c, 70, "Internet radio station not found")
	}

	return utils.SendResponse(c, utils.SuccessResponse())
}
