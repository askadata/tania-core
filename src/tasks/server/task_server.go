package server

import (
	assetsstorage "github.com/Tanibox/tania-server/src/assets/storage"
	cropstorage "github.com/Tanibox/tania-server/src/growth/storage"
	"github.com/Tanibox/tania-server/src/tasks/domain"
	service "github.com/Tanibox/tania-server/src/tasks/domain/service"
	"github.com/Tanibox/tania-server/src/tasks/query/inmemory"
	"github.com/Tanibox/tania-server/src/tasks/repository"
	"github.com/Tanibox/tania-server/src/tasks/storage"
	"github.com/labstack/echo"
	uuid "github.com/satori/go.uuid"
	"net/http"
	"time"
)

// TaskServer ties the routes and handlers with injected dependencies
type TaskServer struct {
	TaskRepo    repository.TaskRepository
	TaskService domain.TaskService
}

// NewTaskServer initializes TaskServer's dependencies and create new TaskServer struct
func NewTaskServer(
	cropStorage *cropstorage.CropStorage,
	areaStorage *assetsstorage.AreaStorage,
	materialStorage *assetsstorage.MaterialStorage) (*TaskServer, error) {

	taskStorage := storage.TaskStorage{TaskMap: make(map[uuid.UUID]domain.Task)}
	taskRepo := repository.NewTaskRepositoryInMemory(&taskStorage)

	cropQuery := inmemory.NewCropQueryInMemory(cropStorage)
	areaQuery := inmemory.NewAreaQueryInMemory(areaStorage)
	materialQuery := inmemory.NewMaterialQueryInMemory(materialStorage)

	taskService := service.TaskServiceInMemory{
		CropQuery:     cropQuery,
		AreaQuery:     areaQuery,
		MaterialQuery: materialQuery,
	}
	return &TaskServer{
		TaskRepo:    taskRepo,
		TaskService: taskService,
	}, nil
}

// Mount defines the TaskServer's endpoints with its handlers
func (s *TaskServer) Mount(g *echo.Group) {
	g.POST("", s.SaveTask)

	g.GET("", s.FindAllTask)
	g.GET("/:id", s.FindTaskByID)
	//g.PUT("/:id/start", s.StartTask)
	g.PUT("/:id/cancel", s.CancelTask)
	g.PUT("/:id/complete", s.CompleteTask)
	// As we don't have an async task right now to check for Due state,
	// I'm adding a rest call to be able to manually do that. We can remove it in the future
	g.PUT("/:id/due", s.SetTaskAsDue)
}

func (s TaskServer) FindAllTask(c echo.Context) error {
	data := make(map[string][]SimpleTask)

	result := <-s.TaskRepo.FindAll()
	if result.Error != nil {
		return result.Error
	}

	Tasks, ok := result.Result.([]domain.Task)
	if !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "Internal server error")
	}

	data["data"] = MapToSimpleTask(Tasks)

	return c.JSON(http.StatusOK, data)
}

// SaveTask is a TaskServer's handler to save new Task
func (s *TaskServer) SaveTask(c echo.Context) error {

	data := make(map[string]domain.Task)

	form_date := c.FormValue("due_date")
	due_ptr := (*time.Time)(nil)
	if len(form_date) != 0 {
		due_date, err := time.Parse(time.RFC3339, form_date)

		if err != nil {
			return Error(c, err)
		}
		due_ptr = &due_date
	}

	asset_id := c.FormValue("asset_id")
	asset_id_ptr := (*uuid.UUID)(nil)
	if len(asset_id) != 0 {
		asset_id, err := uuid.FromString(asset_id)
		if err != nil {
			return Error(c, err)
		}
		asset_id_ptr = &asset_id
	}

	domaintask, err := s.CreateTaskDomainByCode(c)

	if err != nil {
		return Error(c, err)
	}

	task, err := domain.CreateTask(
		s.TaskService,
		c.FormValue("title"),
		c.FormValue("description"),
		due_ptr,
		c.FormValue("priority"),
		domaintask,
		c.FormValue("category"),
		asset_id_ptr)

	if err != nil {
		return Error(c, err)
	}

	err = <-s.TaskRepo.Save(&task)
	if err != nil {
		return Error(c, err)
	}

	data["data"] = task

	return c.JSON(http.StatusOK, data)
}

func (s *TaskServer) CreateTaskDomainByCode(c echo.Context) (domain.TaskDomain, error) {
	domainvalue := c.FormValue("domain")
	if domainvalue == "" {
		return nil, NewRequestValidationError(REQUIRED, "domain")
	}

	switch domainvalue {
	case domain.TaskDomainAreaCode:
		return domain.CreateTaskDomainArea()
	case domain.TaskDomainCropCode:

		uid, err := uuid.FromString(c.FormValue("inventory_id"))
		if err != nil {
			return nil, Error(c, err)
		}
		return domain.CreateTaskDomainCrop(s.TaskService, uid)
	case domain.TaskDomainFinanceCode:
		return domain.CreateTaskDomainFinance()
	case domain.TaskDomainGeneralCode:
		return domain.CreateTaskDomainGeneral()
	case domain.TaskDomainInventoryCode:
		return domain.CreateTaskDomainInventory()
	case domain.TaskDomainReservoirCode:
		return domain.CreateTaskDomainReservoir()
	default:
		return nil, NewRequestValidationError(INVALID_OPTION, "domain")
	}
}

func (s *TaskServer) FindTaskByID(c echo.Context) error {
	data := make(map[string]domain.Task)
	uid, err := uuid.FromString(c.Param("id"))
	if err != nil {
		return Error(c, err)
	}

	result := <-s.TaskRepo.FindByID(c.Param("id"))
	if result.Error != nil {
		return result.Error
	}

	task, ok := result.Result.(domain.Task)

	if task.UID != uid {
		return Error(c, NewRequestValidationError(NOT_FOUND, "id"))
	}
	if !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "Internal server error")
	}

	data["data"] = task

	return c.JSON(http.StatusOK, data)
}

func (s *TaskServer) CancelTask(c echo.Context) error {

	data := make(map[string]domain.Task)
	uid, err := uuid.FromString(c.Param("id"))
	if err != nil {
		return Error(c, err)
	}

	result := <-s.TaskRepo.FindByID(c.Param("id"))
	if result.Error != nil {
		return result.Error
	}

	task, ok := result.Result.(domain.Task)

	if task.UID != uid {
		return Error(c, NewRequestValidationError(NOT_FOUND, "id"))
	}
	if !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "Internal server error")
	}

	task.ChangeTaskStatus(domain.TaskStatusCancelled)

	err = <-s.TaskRepo.Save(&task)
	if err != nil {
		return Error(c, err)
	}

	data["data"] = task

	return c.JSON(http.StatusOK, data)
}

func (s *TaskServer) CompleteTask(c echo.Context) error {

	data := make(map[string]domain.Task)
	uid, err := uuid.FromString(c.Param("id"))
	if err != nil {
		return Error(c, err)
	}

	result := <-s.TaskRepo.FindByID(c.Param("id"))
	if result.Error != nil {
		return result.Error
	}

	task, ok := result.Result.(domain.Task)

	if task.UID != uid {
		return Error(c, NewRequestValidationError(NOT_FOUND, "id"))
	}
	if !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "Internal server error")
	}

	task.ChangeTaskStatus(domain.TaskStatusComplete)

	err = <-s.TaskRepo.Save(&task)
	if err != nil {
		return Error(c, err)
	}

	data["data"] = task

	return c.JSON(http.StatusOK, data)
}

func (s *TaskServer) SetTaskAsDue(c echo.Context) error {

	data := make(map[string]domain.Task)
	uid, err := uuid.FromString(c.Param("id"))
	if err != nil {
		return Error(c, err)
	}

	result := <-s.TaskRepo.FindByID(c.Param("id"))
	if result.Error != nil {
		return result.Error
	}

	task, ok := result.Result.(domain.Task)

	if task.UID != uid {
		return Error(c, NewRequestValidationError(NOT_FOUND, "id"))
	}
	if !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "Internal server error")
	}

	task.SetTaskAsDue()

	err = <-s.TaskRepo.Save(&task)
	if err != nil {
		return Error(c, err)
	}

	data["data"] = task

	return c.JSON(http.StatusOK, data)
}
