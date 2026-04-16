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

// --- Inspection Mutation Responses ---

type inspectionCreateResponse struct {
	InspectionCreate models.Inspection `json:"inspectionCreate"`
}

type inspectionStartResponse struct {
	InspectionStart models.Inspection `json:"inspectionStart"`
}

type inspectionCompleteResponse struct {
	InspectionComplete models.Inspection `json:"inspectionComplete"`
}

type inspectionReopenResponse struct {
	InspectionReopen models.Inspection `json:"inspectionReopen"`
}

type inspectionArchiveResponse struct {
	InspectionArchive models.Inspection `json:"inspectionArchive"`
}

type inspectionExpireResponse struct {
	InspectionExpire models.Inspection `json:"inspectionExpire"`
}

type inspectionUnexpireResponse struct {
	InspectionUnexpire models.Inspection `json:"inspectionUnexpire"`
}

type inspectionSetAssigneeResponse struct {
	InspectionSetAssignee models.Inspection `json:"inspectionSetAssignee"`
}

type inspectionSetDueByResponse struct {
	InspectionSetDueBy models.Inspection `json:"inspectionSetDueBy"`
}

type inspectionSetScheduledForResponse struct {
	InspectionSetScheduledFor models.Inspection `json:"inspectionSetScheduledFor"`
}

type inspectionSetHeaderFieldResponse struct {
	InspectionSetHeaderField models.Inspection `json:"inspectionSetHeaderField"`
}

type inspectionSetFooterFieldResponse struct {
	InspectionSetFooterField models.Inspection `json:"inspectionSetFooterField"`
}

type inspectionSetItemNotesResponse struct {
	InspectionSetItemNotes models.Inspection `json:"inspectionSetItemNotes"`
}

type inspectionRateItemResponse struct {
	InspectionRateItem models.Inspection `json:"inspectionRateItem"`
}

type inspectionAddSectionResponse struct {
	InspectionAddSection models.Inspection `json:"inspectionAddSection"`
}

type inspectionDeleteSectionResponse struct {
	InspectionDeleteSection models.Inspection `json:"inspectionDeleteSection"`
}

type inspectionDuplicateSectionResponse struct {
	InspectionDuplicateSection models.Inspection `json:"inspectionDuplicateSection"`
}

type inspectionRenameSectionResponse struct {
	InspectionRenameSection models.Inspection `json:"inspectionRenameSection"`
}

type inspectionAddItemResponse struct {
	InspectionAddItem models.Inspection `json:"inspectionAddItem"`
}

type inspectionDeleteItemResponse struct {
	InspectionDeleteItem models.Inspection `json:"inspectionDeleteItem"`
}

type inspectionAddItemPhotoResponse struct {
	InspectionAddItemPhoto models.InspectionAddItemPhotoResult `json:"inspectionAddItemPhoto"`
}

type inspectionRemoveItemPhotoResponse struct {
	InspectionRemoveItemPhoto models.Inspection `json:"inspectionRemoveItemPhoto"`
}

type inspectionMoveItemPhotoResponse struct {
	InspectionMoveItemPhoto models.Inspection `json:"inspectionMoveItemPhoto"`
}

type inspectionSendToGuestResponse struct {
	InspectionSendToGuest models.InspectionGuestLink `json:"inspectionSendToGuest"`
}
