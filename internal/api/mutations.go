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

// projectFields is the shared return fragment for project mutations.
const projectFields = `
    id status priority notes start dueAt availabilityTargetAt heldAt createdAt updatedAt
    assignee { id name email }
    location { id name property { id name } }
`

// --- Project Mutations (8) ---

const projectCreateMutation = `mutation ProjectCreate($input: ProjectCreateInput!) {
  projectCreate(input: $input) {` + projectFields + `}
}`

const projectSetAssigneeMutation = `mutation ProjectSetAssignee($input: ProjectSetAssigneeInput!) {
  projectSetAssignee(input: $input) {` + projectFields + `}
}`

const projectSetNotesMutation = `mutation ProjectSetNotes($input: ProjectSetNotesInput!) {
  projectSetNotes(input: $input) {` + projectFields + `}
}`

const projectSetDueAtMutation = `mutation ProjectSetDueAt($input: ProjectSetDueAtInput!) {
  projectSetDueAt(input: $input) {` + projectFields + `}
}`

const projectSetStartAtMutation = `mutation ProjectSetStartAt($input: ProjectSetStartAtInput!) {
  projectSetStartAt(input: $input) {` + projectFields + `}
}`

const projectSetPriorityMutation = `mutation ProjectSetPriority($input: ProjectSetPriorityInput!) {
  projectSetPriority(input: $input) {` + projectFields + `}
}`

const projectSetOnHoldMutation = `mutation ProjectSetOnHold($input: ProjectSetOnHoldInput!) {
  projectSetOnHold(input: $input) {` + projectFields + `}
}`

const projectSetAvailabilityTargetAtMutation = `mutation ProjectSetAvailabilityTargetAt($input: ProjectSetAvailabilityTargetAtInput!) {
  projectSetAvailabilityTargetAt(input: $input) {` + projectFields + `}
}`

// userFields is the shared return fragment for user mutations.
const userFields = `
    id email name shortName phone createdAt updatedAt
`

// membershipFields is the shared return fragment for membership mutations.
const membershipFields = `
    isActive createdAt updatedAt inactivatedAt
    account { id name }
    user { id name email }
    roles(first: 100) { nodes { id name } }
`

// propertyAccessFields is the shared return fragment for property access mutations.
const propertyAccessFields = `
    id accountWideAccess
`

// --- User Mutations (5) ---

const userCreateMutation = `mutation UserCreate($input: UserCreateInput!) {
  userCreate(input: $input) {` + userFields + `}
}`

const userSetEmailMutation = `mutation UserSetEmail($input: UserSetEmailInput!) {
  userSetEmail(input: $input) {` + userFields + `}
}`

const userSetNameMutation = `mutation UserSetName($input: UserSetNameInput!) {
  userSetName(input: $input) {` + userFields + `}
}`

const userSetShortNameMutation = `mutation UserSetShortName($input: UserSetShortNameInput!) {
  userSetShortName(input: $input) {` + userFields + `}
}`

const userSetPhoneMutation = `mutation UserSetPhone($input: UserSetPhoneInput!) {
  userSetPhone(input: $input) {` + userFields + `}
}`

// --- Membership Mutations (4) ---
// Note: PascalCase mutation names match the GraphQL schema.

const accountMembershipCreateMutation = `mutation AccountMembershipCreate($input: AccountMembershipCreateInput!) {
  AccountMembershipCreate(input: $input) {` + membershipFields + `}
}`

const accountMembershipActivateMutation = `mutation AccountMembershipActivate($input: AccountMembershipActivateInput!) {
  AccountMembershipActivate(input: $input) {` + membershipFields + `}
}`

const accountMembershipDeactivateMutation = `mutation AccountMembershipDeactivate($input: AccountMembershipDeactivateInput!) {
  AccountMembershipDeactivate(input: $input) {` + membershipFields + `}
}`

const accountMembershipSetRolesMutation = `mutation AccountMembershipSetRoles($input: AccountMembershipSetRolesInput!) {
  AccountMembershipSetRoles(input: $input) {` + membershipFields + `}
}`

// --- Property Access Mutations (3) ---

const propertyGrantUserAccessMutation = `mutation PropertyGrantUserAccess($input: PropertyGrantUserAccessInput!) {
  PropertyGrantUserAccess(input: $input) {` + propertyAccessFields + `}
}`

const propertyRevokeUserAccessMutation = `mutation PropertyRevokeUserAccess($input: PropertyRevokeUserAccessInput!) {
  PropertyRevokeUserAccess(input: $input) {` + propertyAccessFields + `}
}`

const propertySetAccountWideAccessMutation = `mutation PropertySetAccountWideAccess($input: PropertySetAccountWideAccessInput!) {
  PropertySetAccountWideAccess(input: $input) {` + propertyAccessFields + `}
}`

// --- User Property Access Mutations (2) ---

const userGrantPropertyAccessMutation = `mutation UserGrantPropertyAccess($input: UserGrantPropertyAccessInput!) {
  UserGrantPropertyAccess(input: $input) {` + userFields + `}
}`

const userRevokePropertyAccessMutation = `mutation UserRevokePropertyAccess($input: UserRevokePropertyAccessInput!) {
  UserRevokePropertyAccess(input: $input) {` + userFields + `}
}`

// roleFields is the shared return fragment for role mutations.
const roleFields = `
    id name description createdAt updatedAt archivedAt
    permissions { action description }
`

// --- Role Mutations (4) ---

const roleCreateMutation = `mutation RoleCreate($input: RoleCreateInput!) {
  roleCreate(input: $input) {` + roleFields + `}
}`

const roleSetNameMutation = `mutation RoleSetName($input: RoleSetNameInput!) {
  roleSetName(input: $input) {` + roleFields + `}
}`

const roleSetDescriptionMutation = `mutation RoleSetDescription($input: RoleSetDescriptionInput!) {
  roleSetDescription(input: $input) {` + roleFields + `}
}`

const roleSetPermissionsMutation = `mutation RoleSetPermissions($input: RoleSetPermissionsInput!) {
  roleSetPermissions(input: $input) {` + roleFields + `}
}`

// webhookFields is the shared return fragment for webhook mutations (without signing secret).
const webhookFields = `
    id url status subjects createdAt updatedAt
    subscriber { type id }
    rateLimits { period requests }
    requestTimeout { seconds }
`

// webhookCreateFields includes the signing secret, which is only returned on creation.
const webhookCreateFields = webhookFields + `    signingSecret
`

// --- Webhook Mutations (2) ---

const webhookCreateMutation = `mutation WebhookCreate($input: WebhookCreateInput!) {
  webhookCreate(input: $input) {` + webhookCreateFields + `}
}`

const webhookUpdateMutation = `mutation WebhookUpdate($input: WebhookUpdateInput!) {
  webhookUpdate(input: $input) {` + webhookFields + `}
}`
