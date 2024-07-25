package customer

// // mergeUsers merges users from source customer to target customer
// func (s *Scripts) mergeUsers(ctx context.Context, tx *firestore.Transaction, sourceCustomerRef, targetCustomerRef *firestore.DocumentRef) ([]txUpdateOperations, error) {
// 	l := s.loggerProvider(ctx)
// 	fs := s.conn.Firestore(ctx)

// 	queryer := fs.Collection("users").Where("customer.ref", "==", sourceCustomerRef).Select("customer")

// 	docSnaps, err := tx.Documents(queryer).GetAll()
// 	if err != nil {
// 		return nil, err
// 	}

// 	l.Infof("Found %d users to merge", len(docSnaps))

// 	if len(docSnaps) == 0 {
// 		return nil, nil
// 	}

// 	res := make([]txUpdateOperations, 0, len(docSnaps))

// 	for _, docSnap := range docSnaps {
// 		res = append(res, txUpdateOperations{
// 			ref:     docSnap.Ref,
// 			updates: []firestore.Update{{Path: "customer", Value: targetCustomerRef}},
// 		})
// 	}

// 	return res, nil
// }
