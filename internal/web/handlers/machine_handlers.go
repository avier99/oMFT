package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/avier99/oMFT/components"
	"github.com/avier99/oMFT/internal/db"
)

func (h *Handlers) HandleMachines(c *gin.Context) {
	userID := c.GetUint("userID")
	machines, err := h.DB.GetMachines(userID)
	if err != nil {
		log.Printf("Error loading machines: %v", err)
		machines = []db.Machine{}
	}
	components.Machines(c.Request.Context(), components.MachinesData{Machines: machines}).Render(c, c.Writer)
}

func (h *Handlers) HandleNewMachine(c *gin.Context) {
	components.MachineForm(c.Request.Context(), components.MachineFormData{
		Machine: &db.Machine{},
		IsNew:   true,
	}).Render(c, c.Writer)
}

func (h *Handlers) HandleEditMachine(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.Redirect(http.StatusFound, "/machines")
		return
	}
	machine, err := h.DB.GetMachine(uint(id))
	if err != nil {
		c.Redirect(http.StatusFound, "/machines")
		return
	}
	userID := c.GetUint("userID")
	if machine.CreatedBy != userID {
		isAdmin, _ := c.Get("isAdmin")
		if isAdmin != true {
			c.Redirect(http.StatusFound, "/machines")
			return
		}
	}
	components.MachineForm(c.Request.Context(), components.MachineFormData{
		Machine: machine,
		IsNew:   false,
	}).Render(c, c.Writer)
}

func (h *Handlers) HandleCreateMachine(c *gin.Context) {
	var machine db.Machine
	if err := c.ShouldBind(&machine); err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("Invalid form data: %v", err))
		return
	}
	passiveModeVal := c.Request.FormValue("passive_mode")
	passiveModeValue := passiveModeVal == "on" || passiveModeVal == "true"
	machine.PassiveMode = &passiveModeValue

	userID := c.GetUint("userID")
	machine.CreatedBy = userID

	if err := h.DB.CreateMachine(&machine); err != nil {
		c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to create machine: %v", err))
		return
	}

	hasSecrets := machine.Password != "" || machine.SecretKey != "" || machine.ClientSecret != ""
	if hasSecrets {
		if err := h.DB.GenerateMachineRcloneConfig(&machine); err != nil {
			log.Printf("Warning: failed to generate machine rclone config for machine %d: %v", machine.ID, err)
		}
	}

	c.Redirect(http.StatusFound, "/machines")
}

func (h *Handlers) HandleUpdateMachine(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid ID")
		return
	}
	existing, err := h.DB.GetMachine(uint(id))
	if err != nil {
		c.String(http.StatusNotFound, "Machine not found")
		return
	}
	userID := c.GetUint("userID")
	if existing.CreatedBy != userID {
		isAdmin, _ := c.Get("isAdmin")
		if isAdmin != true {
			c.String(http.StatusForbidden, "Permission denied")
			return
		}
	}

	var machine db.Machine
	if err := c.ShouldBind(&machine); err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("Invalid form data: %v", err))
		return
	}
	passiveModeVal := c.Request.FormValue("passive_mode")
	passiveModeValue := passiveModeVal == "on" || passiveModeVal == "true"
	machine.PassiveMode = &passiveModeValue

	machine.ID = existing.ID
	machine.CreatedBy = existing.CreatedBy
	machine.CreatedAt = existing.CreatedAt

	if err := h.DB.UpdateMachine(&machine); err != nil {
		c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to update machine: %v", err))
		return
	}

	hasSecrets := machine.Password != "" || machine.SecretKey != "" || machine.ClientSecret != ""
	if hasSecrets {
		if err := h.DB.GenerateMachineRcloneConfig(&machine); err != nil {
			log.Printf("Warning: failed to regenerate machine rclone config for machine %d: %v", machine.ID, err)
		}
	}

	c.Redirect(http.StatusSeeOther, "/machines")
}

func (h *Handlers) HandleDeleteMachine(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}
	machine, err := h.DB.GetMachine(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Machine not found"})
		return
	}
	userID := c.GetUint("userID")
	if machine.CreatedBy != userID {
		isAdmin, _ := c.Get("isAdmin")
		if isAdmin != true {
			c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
			return
		}
	}
	if err := h.DB.DeleteMachine(uint(id)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Machine deleted"})
}

func (h *Handlers) HandleTestMachineConnection(c *gin.Context) {
	var machine db.Machine
	if err := c.ShouldBind(&machine); err != nil {
		sendMachineTestToast(c, false, fmt.Sprintf("Invalid form data: %v", err))
		return
	}
	passiveModeVal := c.Request.FormValue("passive_mode")
	passiveModeValue := passiveModeVal == "on" || passiveModeVal == "true"
	machine.PassiveMode = &passiveModeValue

	testPath := c.PostForm("test_path")

	_, message, _ := h.DB.TestMachineConnection(&machine, testPath)
	success := message == "Connection test successful!"
	sendMachineTestToast(c, success, message)
}

func sendMachineTestToast(c *gin.Context, success bool, message string) {
	toastType := "error"
	if success {
		toastType = "success"
	}
	data, err := json.Marshal(map[string]interface{}{
		"showToast": map[string]string{
			"message": message,
			"type":    toastType,
		},
	})
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	c.Header("HX-Trigger", string(data))
	c.Status(http.StatusOK)
}
