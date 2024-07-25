package service

import (
	"context"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/customerapi"
	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
)

func (s *BudgetsService) GetBudgetExternal(ctx context.Context, budgetID string, email string, customerID string) (*BudgetAPI, error) {
	internalBudget, err := s.dal.GetBudget(ctx, budgetID)
	if err == doitFirestore.ErrNotFound {
		return nil, web.ErrNotFound
	}

	if err != nil {
		return nil, err
	}

	if internalBudget.Customer.ID != customerID {
		return nil, web.ErrUnauthorized
	}

	if internalBudget.Public == nil && !isEmailOnCollaborators(email, internalBudget.Collaborators) {
		return nil, web.ErrUnauthorized
	}

	internalBudget.ID = budgetID

	responseBudget, err := mapInternalBudgetToResponseBudget(internalBudget)
	if err != nil {
		return nil, err
	}

	return responseBudget, nil
}

func (s *BudgetsService) ListBudgets(ctx context.Context, args *ExternalAPIListArgsReq) (bl *BudgetList, paramsError error, internalError error) {
	listBudgetArgs, err := validateListBudgetsArgs(args)
	if err != nil {
		return nil, err, nil
	}

	budgets, err := s.dal.ListBudgets(ctx, listBudgetArgs)
	if err != nil {
		return nil, nil, err
	}

	apiBudgets := toListBudgetsAPI(budgets, args.CustomerID, listBudgetArgs.Filter.Owners, listBudgetArgs.Filter.TimeModified)

	sortedBudgets, err := customerapi.SortAPIList(apiBudgets, listBudgetArgs.Filter.OrderBy, firestore.Desc)
	if err != nil {
		return nil, nil, err
	}

	page, token, err := customerapi.GetEncodedAPIPage(listBudgetArgs.MaxResults, listBudgetArgs.PageToken, sortedBudgets)
	if err != nil {
		return nil, nil, err
	}

	budgetList := &BudgetList{
		Budgets:   page,
		PageToken: token,
		RowCount:  len(page),
	}

	return budgetList, nil, nil
}
