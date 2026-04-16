package api

// workOrderFields is the shared return fragment for work order mutations.
// Matches the fields returned by listWorkOrdersQuery for consistency.
const workOrderFields = `
    id status subStatus description summary priority createdAt updatedAt scheduledFor
    assignedTo { ... on WorkOrderAssigneeUser { __typename id name email type } ... on WorkOrderAssigneeVendor { __typename id name email type } }
    locationV2 { id name property { id name } }
    inspectionDetails { inspection { id name status } }
`

// --- Work Order Mutations (19) ---

const workOrderCreateMutation = `mutation WorkOrderCreate($input: WorkOrderCreateInput!) {
  workOrderCreate(input: $input) {` + workOrderFields + `}
}`

const workOrderSetStatusAndSubStatusMutation = `mutation WorkOrderSetStatusAndSubStatus($input: WorkOrderSetStatusAndSubStatusInput) {
  workOrderSetStatusAndSubStatus(input: $input) {` + workOrderFields + `}
}`

const workOrderSetAssigneeMutation = `mutation WorkOrderSetAssignee($input: WorkOrderSetAssigneeInput!) {
  workOrderSetAssignee(input: $input) {` + workOrderFields + `}
}`

const workOrderSetDescriptionMutation = `mutation WorkOrderSetDescription($input: WorkOrderSetDescriptionInput!) {
  workOrderSetDescription(input: $input) {` + workOrderFields + `}
}`

const workOrderSetPriorityMutation = `mutation WorkOrderSetPriority($input: WorkOrderSetPriorityInput!) {
  workOrderSetPriority(input: $input) {` + workOrderFields + `}
}`

const workOrderSetScheduledForMutation = `mutation WorkOrderSetScheduledFor($input: WorkOrderSetScheduledForInput!) {
  workOrderSetScheduledFor(input: $input) {` + workOrderFields + `}
}`

const workOrderSetLocationMutation = `mutation WorkOrderSetLocation($input: WorkOrderSetLocationInput!) {
  workOrderSetLocation(input: $input) {` + workOrderFields + `}
}`

const workOrderSetTypeMutation = `mutation WorkOrderSetType($input: WorkOrderSetTypeInput!) {
  workOrderSetType(input: $input) {` + workOrderFields + `}
}`

const workOrderSetEntryNotesMutation = `mutation WorkOrderSetEntryNotes($input: WorkOrderSetEntryNotesInput!) {
  workOrderSetEntryNotes(input: $input) {` + workOrderFields + `}
}`

const workOrderSetPermissionToEnterMutation = `mutation WorkOrderSetPermissionToEnter($input: WorkOrderSetPermissionToEnterInput!) {
  workOrderSetPermissionToEnter(input: $input) {` + workOrderFields + `}
}`

const workOrderSetResidentApprovedEntryMutation = `mutation WorkOrderSetResidentApprovedEntry($input: WorkOrderSetResidentApprovedEntryInput!) {
  workOrderSetResidentApprovedEntry(input: $input) {` + workOrderFields + `}
}`

const workOrderSetUnitEnteredMutation = `mutation WorkOrderSetUnitEntered($input: WorkOrderSetUnitEnteredInput!) {
  workOrderSetUnitEntered(input: $input) {` + workOrderFields + `}
}`

const workOrderArchiveMutation = `mutation WorkOrderArchive($input: WorkOrderArchiveInput!) {
  workOrderArchive(input: $input) {` + workOrderFields + `}
}`

const workOrderAddCommentMutation = `mutation WorkOrderAddComment($input: WorkOrderAddCommentInput!) {
  workOrderAddComment(input: $input) {` + workOrderFields + `}
}`

const workOrderAddTimeMutation = `mutation WorkOrderAddTime($input: WorkOrderAddTimeInput!) {
  workOrderAddTime(input: $input) {` + workOrderFields + `}
}`

const workOrderAddAttachmentMutation = `mutation WorkOrderAddAttachment($input: WorkOrderAddAttachmentInput!) {
  workOrderAddAttachment(input: $input) {
    workOrder {` + workOrderFields + `}
    attachment { id name mediaType }
    signedURL
  }
}`

const workOrderRemoveAttachmentMutation = `mutation WorkOrderRemoveAttachment($input: WorkOrderRemoveAttachmentInput!) {
  workOrderRemoveAttachment(input: $input) {` + workOrderFields + `}
}`

const workOrderStartTimerMutation = `mutation WorkOrderStartTimer($input: WorkOrderStartTimerInput!) {
  workOrderStartTimer(input: $input) {` + workOrderFields + `}
}`

const workOrderStopTimerMutation = `mutation WorkOrderStopTimer($input: WorkOrderStopTimerInput!) {
  workOrderStopTimer(input: $input) {` + workOrderFields + `}
}`

// inspectionFields is the shared return fragment for inspection mutations.
// Matches the fields returned by listInspectionsQuery for consistency.
const inspectionFields = `
    id name status startedAt endedAt scheduledFor dueBy score potentialScore
    assignedTo { ... on InspectionAssigneeUser { __typename id name email type } }
    locationV2 { id name property { id name } }
    templateV2 { id name }
`

// --- Inspection Mutations (24) ---

const inspectionCreateMutation = `mutation InspectionCreate($input: InspectionCreateInput!) {
  inspectionCreate(input: $input) {` + inspectionFields + `}
}`

const inspectionStartMutation = `mutation InspectionStart($input: InspectionStartInput!) {
  inspectionStart(input: $input) {` + inspectionFields + `}
}`

const inspectionCompleteMutation = `mutation InspectionComplete($input: InspectionCompleteInput!) {
  inspectionComplete(input: $input) {` + inspectionFields + `}
}`

const inspectionReopenMutation = `mutation InspectionReopen($input: InspectionReopenInput!) {
  inspectionReopen(input: $input) {` + inspectionFields + `}
}`

const inspectionArchiveMutation = `mutation InspectionArchive($input: InspectionArchiveInput!) {
  inspectionArchive(input: $input) {` + inspectionFields + `}
}`

const inspectionExpireMutation = `mutation InspectionExpire($input: InspectionExpireInput!) {
  inspectionExpire(input: $input) {` + inspectionFields + `}
}`

const inspectionUnexpireMutation = `mutation InspectionUnexpire($input: InspectionUnexpireInput!) {
  inspectionUnexpire(input: $input) {` + inspectionFields + `}
}`

const inspectionSetAssigneeMutation = `mutation InspectionSetAssignee($input: InspectionSetAssigneeInput!) {
  inspectionSetAssignee(input: $input) {` + inspectionFields + `}
}`

const inspectionSetDueByMutation = `mutation InspectionSetDueBy($input: InspectionSetDueByInput!) {
  inspectionSetDueBy(input: $input) {` + inspectionFields + `}
}`

const inspectionSetScheduledForMutation = `mutation InspectionSetScheduledFor($input: InspectionSetScheduledForInput!) {
  inspectionSetScheduledFor(input: $input) {` + inspectionFields + `}
}`

const inspectionSetHeaderFieldMutation = `mutation InspectionSetHeaderField($input: InspectionSetHeaderFieldInput!) {
  inspectionSetHeaderField(input: $input) {` + inspectionFields + `}
}`

const inspectionSetFooterFieldMutation = `mutation InspectionSetFooterField($input: InspectionSetFooterFieldInput!) {
  inspectionSetFooterField(input: $input) {` + inspectionFields + `}
}`

const inspectionSetItemNotesMutation = `mutation InspectionSetItemNotes($input: InspectionSetItemNotesInput!) {
  inspectionSetItemNotes(input: $input) {` + inspectionFields + `}
}`

const inspectionRateItemMutation = `mutation InspectionRateItem($input: InspectionRateItemInput!) {
  inspectionRateItem(input: $input) {` + inspectionFields + `}
}`

const inspectionAddSectionMutation = `mutation InspectionAddSection($input: InspectionAddSectionInput!) {
  inspectionAddSection(input: $input) {` + inspectionFields + `}
}`

const inspectionDeleteSectionMutation = `mutation InspectionDeleteSection($input: InspectionDeleteSectionInput!) {
  inspectionDeleteSection(input: $input) {` + inspectionFields + `}
}`

const inspectionDuplicateSectionMutation = `mutation InspectionDuplicateSection($input: InspectionDuplicateSectionInput!) {
  inspectionDuplicateSection(input: $input) {` + inspectionFields + `}
}`

const inspectionRenameSectionMutation = `mutation InspectionRenameSection($input: InspectionRenameSectionInput!) {
  inspectionRenameSection(input: $input) {` + inspectionFields + `}
}`

const inspectionAddItemMutation = `mutation InspectionAddItem($input: InspectionAddItemInput!) {
  inspectionAddItem(input: $input) {` + inspectionFields + `}
}`

const inspectionDeleteItemMutation = `mutation InspectionDeleteItem($input: InspectionDeleteItemInput!) {
  inspectionDeleteItem(input: $input) {` + inspectionFields + `}
}`

const inspectionAddItemPhotoMutation = `mutation InspectionAddItemPhoto($input: InspectionAddItemPhotoInput!) {
  inspectionAddItemPhoto(input: $input) {
    inspection {` + inspectionFields + `}
    inspectionPhoto { id mimeType }
    signedURL
  }
}`

const inspectionRemoveItemPhotoMutation = `mutation InspectionRemoveItemPhoto($input: InspectionRemoveItemPhotoInput!) {
  inspectionRemoveItemPhoto(input: $input) {` + inspectionFields + `}
}`

const inspectionMoveItemPhotoMutation = `mutation InspectionMoveItemPhoto($input: InspectionMoveItemPhotoInput!) {
  inspectionMoveItemPhoto(input: $input) {` + inspectionFields + `}
}`

const inspectionSendToGuestMutation = `mutation InspectionSendToGuest($input: InspectionSendToGuestInput!) {
  inspectionSendToGuest(input: $input) {
    inspectionId
    link
  }
}`
