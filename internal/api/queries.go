package api

const loginMutation = `mutation Login($input: LoginInput!) {
  login(input: $input) { token expiresAt accessibleBusinessIds }
}`

const getAccountQuery = `query GetAccount($accountId: ID!) {
  account(accountId: $accountId) { id name }
}`

const listPropertiesQuery = `query ListProperties($accountId: ID!, $first: Int, $after: String, $filter: AccountPropertiesFilter, $orderBy: [PropertyV2OrderBy!]) {
  account(accountId: $accountId) {
    properties(first: $first, after: $after, filter: $filter, orderBy: $orderBy) {
      count
      pageInfo { hasNextPage endCursor }
      edges { cursor node { id name createdAt address { line1 line2 city state country postalCode } } }
    }
  }
}`

const listUnitsQuery = `query ListUnits($accountId: ID!, $propertiesFilter: AccountPropertiesFilter, $first: Int, $after: String) {
  account(accountId: $accountId) {
    properties(filter: $propertiesFilter) {
      edges { node {
        units(first: $first, after: $after) {
          count
          pageInfo { hasNextPage endCursor }
          edges { cursor node { id name } }
        }
      } }
    }
  }
}`

const listWorkOrdersQuery = `query ListWorkOrders($accountId: ID!, $first: Int, $after: String, $filter: AccountWorkOrderFilter, $orderBy: [WorkOrderOrderBy!]) {
  account(accountId: $accountId) {
    workOrders(first: $first, after: $after, filter: $filter, orderBy: $orderBy) {
      count
      pageInfo { hasNextPage endCursor }
      edges { cursor node {
        id status subStatus description summary priority createdAt updatedAt scheduledFor
        assignedTo { ... on WorkOrderAssigneeUser { __typename id name email type } ... on WorkOrderAssigneeVendor { __typename id name email type } }
        locationV2 { id name property { id name } }
        inspectionDetails { inspection { id name status } }
      } }
    }
  }
}`

const listInspectionsQuery = `query ListInspections($accountId: ID!, $first: Int, $after: String, $filter: AccountInspectionFilter, $orderBy: [InspectionOrderBy!]) {
  account(accountId: $accountId) {
    inspections(first: $first, after: $after, filter: $filter, orderBy: $orderBy) {
      count
      pageInfo { hasNextPage endCursor }
      edges { cursor node {
        id name status startedAt endedAt scheduledFor dueBy score potentialScore
        assignedTo { ... on InspectionAssigneeUser { __typename id name email type } }
        locationV2 { id name property { id name } }
        templateV2 { id name }
      } }
    }
  }
}`
