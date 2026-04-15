package api

import "github.com/findingsimple/hppy-connect/internal/models"

// --- Work Order Mutation Responses ---

// Most work order mutations return a WorkOrder directly at the top level.
// Shared response type for single-entity returns.

type workOrderCreateResponse struct {
	WorkOrderCreate models.WorkOrder `json:"workOrderCreate"`
}

type workOrderSetStatusAndSubStatusResponse struct {
	WorkOrderSetStatusAndSubStatus models.WorkOrder `json:"workOrderSetStatusAndSubStatus"`
}

type workOrderSetAssigneeResponse struct {
	WorkOrderSetAssignee models.WorkOrder `json:"workOrderSetAssignee"`
}

type workOrderSetDescriptionResponse struct {
	WorkOrderSetDescription models.WorkOrder `json:"workOrderSetDescription"`
}

type workOrderSetPriorityResponse struct {
	WorkOrderSetPriority models.WorkOrder `json:"workOrderSetPriority"`
}

type workOrderSetScheduledForResponse struct {
	WorkOrderSetScheduledFor models.WorkOrder `json:"workOrderSetScheduledFor"`
}

type workOrderSetLocationResponse struct {
	WorkOrderSetLocation models.WorkOrder `json:"workOrderSetLocation"`
}

type workOrderSetTypeResponse struct {
	WorkOrderSetType models.WorkOrder `json:"workOrderSetType"`
}

type workOrderSetEntryNotesResponse struct {
	WorkOrderSetEntryNotes models.WorkOrder `json:"workOrderSetEntryNotes"`
}

type workOrderSetPermissionToEnterResponse struct {
	WorkOrderSetPermissionToEnter models.WorkOrder `json:"workOrderSetPermissionToEnter"`
}

type workOrderSetResidentApprovedEntryResponse struct {
	WorkOrderSetResidentApprovedEntry models.WorkOrder `json:"workOrderSetResidentApprovedEntry"`
}

type workOrderSetUnitEnteredResponse struct {
	WorkOrderSetUnitEntered models.WorkOrder `json:"workOrderSetUnitEntered"`
}

type workOrderArchiveResponse struct {
	WorkOrderArchive models.WorkOrder `json:"workOrderArchive"`
}

type workOrderAddCommentResponse struct {
	WorkOrderAddComment models.WorkOrder `json:"workOrderAddComment"`
}

type workOrderAddTimeResponse struct {
	WorkOrderAddTime models.WorkOrder `json:"workOrderAddTime"`
}

// workOrderAddAttachmentResponse has a multi-field return shape.
type workOrderAddAttachmentResponse struct {
	WorkOrderAddAttachment models.WorkOrderAddAttachmentResult `json:"workOrderAddAttachment"`
}

type workOrderRemoveAttachmentResponse struct {
	WorkOrderRemoveAttachment models.WorkOrder `json:"workOrderRemoveAttachment"`
}

type workOrderStartTimerResponse struct {
	WorkOrderStartTimer models.WorkOrder `json:"workOrderStartTimer"`
}

type workOrderStopTimerResponse struct {
	WorkOrderStopTimer models.WorkOrder `json:"workOrderStopTimer"`
}
