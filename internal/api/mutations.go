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
