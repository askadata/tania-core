package domain

import (
	"github.com/Tanibox/tania-server/src/tasks/query"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"
)

type TaskServiceMock struct {
	mock.Mock
}

func (m TaskServiceMock) FindAreaByID(uid uuid.UUID) ServiceResult {
	args := m.Called(uid)
	return args.Get(0).(ServiceResult)
}
func (m TaskServiceMock) FindCropByID(uid uuid.UUID) ServiceResult {
	args := m.Called(uid)
	return args.Get(0).(ServiceResult)
}
func (m TaskServiceMock) FindMaterialByID(uid uuid.UUID) ServiceResult {
	args := m.Called(uid)
	return args.Get(0).(ServiceResult)
}

func TestCreateTask(t *testing.T) {
	taskServiceMock := new(TaskServiceMock)

	assetID, _ := uuid.NewV4()
	assetID_notexist, _ := uuid.NewV4()

	due_date_invalid, _ := time.Parse(time.RFC3339, "2017-01-23T17:37:39.697328206+01:00")
	due_ptr_invalid := &due_date_invalid
	due_date, _ := time.Parse(time.RFC3339, "2019-01-23T17:37:39.697328206+01:00")
	due_ptr := &due_date

	tasktitle := "My Task"
	taskdescription := "My Description"
	taskcategory := "SANITATION"

	inventoryID, _ := uuid.NewV4()

	taskServiceMock.On("FindMaterialByID", inventoryID).Return(ServiceResult{Result: query.TaskMaterialQueryResult{UID: inventoryID}})
	taskdomain, _ := CreateTaskDomainCrop(taskServiceMock, inventoryID)

	var tests = []struct {
		title              string
		description        string
		duedate            *time.Time
		priority           string
		domain             TaskDomain
		category           string
		assetid            *uuid.UUID
		eexpectedTaskError error
	}{
		//empty title
		{"", taskdescription, due_ptr, "NORMAL", taskdomain, taskcategory, &assetID, TaskError{TaskErrorTitleEmptyCode}},
		//emptyduedate
		{tasktitle, taskdescription, nil, "URGENT", taskdomain, taskcategory, &assetID, nil},
		//duedatepassed
		{tasktitle, taskdescription, due_ptr_invalid, "NORMAL", taskdomain, taskcategory, &assetID, TaskError{TaskErrorDueDateInvalidCode}},
		//empty priority
		{tasktitle, taskdescription, due_ptr, "", taskdomain, taskcategory, &assetID, TaskError{TaskErrorPriorityEmptyCode}},
		//invalidpriority
		{tasktitle, taskdescription, due_ptr, "urgent", taskdomain, taskcategory, &assetID, TaskError{TaskErrorInvalidPriorityCode}},
		//empty category
		{tasktitle, taskdescription, due_ptr, "URGENT", taskdomain, "", &assetID, TaskError{TaskErrorCategoryEmptyCode}},
		//invalid category
		{tasktitle, taskdescription, due_ptr, "NORMAL", taskdomain, "VEGETABLE", &assetID, TaskError{TaskErrorInvalidCategoryCode}},
	}

	for _, test := range tests {
		taskServiceMock.On("FindCropByID", *test.assetid).Return(ServiceResult{Result: query.TaskCropQueryResult{}})
		taskServiceMock.On("FindMaterialByID", inventoryID).Return(ServiceResult{Result: query.TaskMaterialQueryResult{UID: inventoryID}})

		_, err := CreateTask(
			taskServiceMock, test.title, test.description, test.duedate, test.priority, test.domain, test.category, test.assetid)

		assert.Equal(t, test.eexpectedTaskError, err)
	}

	//nil assetid
	taskServiceMock.On("FindMaterialByID", inventoryID).Return(ServiceResult{Result: query.TaskMaterialQueryResult{UID: inventoryID}})

	_, err := CreateTask(
		taskServiceMock, tasktitle, taskdescription, due_ptr, "URGENT", taskdomain, taskcategory, nil)

	assert.Equal(t, nil, err)

	//assetid doesn't exist
	taskServiceMock.On("FindCropByID", assetID_notexist).Return(ServiceResult{Result: query.TaskCropQueryResult{}, Error: TaskError{TaskErrorInvalidAssetIDCode}})
	taskServiceMock.On("FindMaterialByID", inventoryID).Return(ServiceResult{Result: query.TaskMaterialQueryResult{UID: inventoryID}})

	_, err = CreateTask(
		taskServiceMock, tasktitle, taskdescription, due_ptr, "NORMAL", taskdomain, taskcategory, &assetID_notexist)

	assert.Equal(t, TaskError{TaskErrorInvalidAssetIDCode}, err)
}
