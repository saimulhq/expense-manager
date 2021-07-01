// Simple CRUD Application using golang, grpc, grpc-gateway & mongoDB

package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"

	"expense-manager/expensepb"
	"expense-manager/util"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// global variable for db collection
var collection *mongo.Collection

// set server of type ExpenseService
type server struct {
	expensepb.UnimplementedExpenseServiceServer
}

// expense struct which can be converted to bson
type expense struct {
	Id          primitive.ObjectID `bson:"_id,omitempty"`
	Title       string             `bson:"title"`
	Description string             `bson:"descrition"`
	Amount      int32              `bson:"amount"`
	Price       int32              `bson:"price"`
	Date        string             `bson:"date"`
}

// serving Swagger UI for REST calls using swagger.json file
func serveSwagger(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./expensepb/expense.swagger.json")
}

// receiver function for creating a new expense
func (*server) CreateExpense(ctx context.Context, req *expensepb.CreateExpenseRequest) (*expensepb.CreateExpenseResponse, error) {
	// get parameters from request
	title := req.Expense.Title
	description := req.Expense.Description
	amount := req.Expense.Amount
	price := req.Expense.Price
	date := req.Expense.Date

	// assigning the request params to data struct of type expense
	data := expense{
		Title:       title,
		Description: description,
		Amount:      amount,
		Price:       price,
		Date:        date,
	}

	// insering data to db
	res, err := collection.InsertOne(context.Background(), data)
	if err != nil {
		// return internal server error if error occurs
		return nil, status.Errorf(codes.Internal, fmt.Sprintln("Internal error: ", err))
	}
	// cast to object id
	oid, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		// return internal server error if error occurs
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintln("Cannot convert to Object ID"),
		)
	}

	// log when an expense is created
	log.Println("Created an expense with id: " + oid.Hex())

	// return response for createExpense if everything's okay
	return &expensepb.CreateExpenseResponse{
		Id: oid.Hex(), // convert object id to string
		Expense: &expensepb.Expense{
			Title:       data.Title,
			Description: data.Description,
			Amount:      data.Amount,
			Price:       data.Price,
			Date:        data.Date,
		},
	}, nil
}

// receiver function for getting an expense by id
func (*server) GetExpense(ctx context.Context, req *expensepb.GetExpenseRequest) (*expensepb.GetExpenseResponse, error) {
	// get id from request
	expenseId := req.Id

	// cast to object id
	oid, err := primitive.ObjectIDFromHex(expenseId)
	if err != nil {
		// return invalid argument error if error occurs
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintln("Cannot parse ID"))
	}

	// creating an empty expense struct
	data := &expense{}

	// creating a filter for finding data with object id
	filter := bson.M{"_id": oid}

	// finding data with object id
	res := collection.FindOne(context.Background(), filter)
	if err := res.Decode(data); err != nil {
		// return not found error if error occurs
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("Cannot find expense with the specified ID: %v", err))
	}

	// log when fetch an expense with id
	log.Println("Fetched an expense with id: " + data.Id.Hex())

	// return response for getExpense if everything's okay
	return &expensepb.GetExpenseResponse{
		Id: data.Id.Hex(), // convert object id to string
		Expense: &expensepb.Expense{
			Title:       data.Title,
			Description: data.Description,
			Amount:      data.Amount,
			Price:       data.Price,
			Date:        data.Date,
		},
	}, nil
}

// receiver function for updating an expense by id
func (*server) UpdateExpense(ctx context.Context, req *expensepb.UpdateExpenseRequest) (*expensepb.UpdateExpenseResponse, error) {
	// get parameters from request
	requestId := req.Id
	requestedExpense := req.Expense

	// cast to object id
	oid, err := primitive.ObjectIDFromHex(requestId)
	if err != nil {
		// return invalid argument error if error occurs
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintln("Cannot parse ID"))
	}

	data := &expense{}

	filter := bson.M{"_id": oid}

	res := collection.FindOne(context.Background(), filter)
	if err := res.Decode(data); err != nil {
		// return not found error if error occurs
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("Cannot find expense with the specified ID: %v", err))
	}

	// update data with changes from request if any
	data.Title = requestedExpense.Title
	data.Description = requestedExpense.Description
	data.Amount = requestedExpense.Amount
	data.Price = requestedExpense.Price
	data.Date = requestedExpense.Date

	// update data in db
	_, updateErr := collection.ReplaceOne(context.Background(), filter, data)
	if updateErr != nil {
		// return internal server error if error occurs
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("Cannot update expense in database: %v", updateErr))
	}

	// log when update an expense with id
	log.Println("Updated an expense with id: " + oid.Hex())

	// return response for updateExpense if everything's okay
	return &expensepb.UpdateExpenseResponse{
		Id: data.Id.Hex(),
		Expense: &expensepb.Expense{
			Title:       data.Title,
			Description: data.Description,
			Amount:      data.Amount,
			Price:       data.Price,
			Date:        data.Date,
		},
	}, nil
}

// receiver function for deleting an expense by id
func (*server) DeleteExpense(ctx context.Context, req *expensepb.DeleteExpenseRequest) (*expensepb.DeleteExpenseResponse, error) {
	// get id from request
	requestId := req.Id

	// cast to object id
	oid, err := primitive.ObjectIDFromHex(requestId)
	if err != nil {
		// return invalid argument error if error occurs
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintln("Cannot parse ID"))
	}

	filter := bson.M{"_id": oid}

	// delete data from db
	res, err := collection.DeleteOne(context.Background(), filter)
	if err != nil {
		// return internal server error if error occurs
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("Cannot delete expense with the specified ID: %v", err))
	}

	// if response deletedCount is zero return not found error
	if res.DeletedCount == 0 {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("Cannot find expense with the specified ID: %v", err))
	}

	// log when delete an expense with id
	log.Println("Deleted an expense with id: " + oid.Hex())

	// return id of the data deleted in response
	return &expensepb.DeleteExpenseResponse{
		Id: requestId,
	}, nil
}

// receiver function for getting all the expenses
func (*server) GetAllExpense(ctx context.Context, req *expensepb.GetAllExpenseRequest) (*expensepb.GetAllExpenseReponse, error) {
	// find data from db
	cur, err := collection.Find(context.Background(), primitive.D{{}})
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("Unknown internal error: %v", err))
	}

	// close curosr when done
	defer cur.Close(context.Background())
	// creating an empty expense list
	var expenseList = []*expensepb.ExpenseWithId{}
	// loop cursor one by one
	for cur.Next(context.Background()) {
		data := &expense{}
		err := cur.Decode(data)
		if err != nil {
			return nil, status.Errorf(codes.Internal, fmt.Sprintf("Error while decoding data: %v", err))
		}

		// create data of type expenseWithId
		convertedData := expensepb.ExpenseWithId{
			Id:          data.Id.Hex(),
			Title:       data.Title,
			Description: data.Description,
			Amount:      data.Amount,
			Price:       data.Price,
			Date:        data.Date,
		}
		// append newly created data to the expense list
		expenseList = append(expenseList, &convertedData)
	}

	// if cursor provides an error return in response
	if err := cur.Err(); err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("Unknown internal error: %v", err))
	}

	// log when get all expenses
	log.Println("Fetched all expenses")

	// return list of all expenses in response
	return &expensepb.GetAllExpenseReponse{
		Expense: expenseList}, nil
}

// main function
func main() {
	// load configurations
	config, err := util.LoadConfig("/expense-manager/")
	if err != nil {
		log.Fatalln("Cannot load config: ", err)
	}

	// connect to db
	client, err := mongo.NewClient(options.Client().ApplyURI(config.DBUrl))
	if err != nil {
		log.Fatalln(err)
	}
	err = client.Connect(context.TODO())
	if err != nil {
		log.Fatalln(err)
	}

	// configure the collection with db name and collection name
	collection = client.Database(config.DBName).Collection(config.DBCollectionName)

	log.Println("Connected to database")

	// grpc address string
	grpcAddress := config.GrpcHost + ":" + config.GrpcPort
	// http address string
	httpAddress := config.HttpHost + ":" + config.HttpPort

	// create a grpc listener on TCP port
	lis, err := net.Listen("tcp", ":"+config.GrpcPort)
	if err != nil {
		log.Fatalln("Failed to listen: ", err)
	}

	// creating a gRPC server object
	s := grpc.NewServer()
	// registering the expense service to the server
	expensepb.RegisterExpenseServiceServer(s, &server{})

	log.Println("Serving gRPC on " + grpcAddress)
	// serving the gRPC server in a child routine
	go func() {
		log.Fatalln(s.Serve(lis))
	}()

	// creating a client connection to the gRPC server
	// this is where the gRPC-Gateway proxies the requests
	conn, err := grpc.DialContext(
		context.Background(),
		grpcAddress,
		grpc.WithBlock(),
		grpc.WithInsecure(),
	)
	if err != nil {
		log.Fatalln("Failed to dial server:  ", err)
	}

	gwmux := runtime.NewServeMux()
	// register Expense Service Handler
	err = expensepb.RegisterExpenseServiceHandler(context.Background(), gwmux, conn)
	if err != nil {
		log.Fatalln("Failed to register gateway: ", err)
	}

	// serving the swagger-ui and swagger file
	mux := http.NewServeMux()
	mux.Handle("/", gwmux)
	mux.HandleFunc("/swagger.json", serveSwagger)
	fileServer := http.FileServer(http.Dir("swagger-ui"))
	mux.Handle("/swagger-ui/", http.StripPrefix("/swagger-ui", fileServer))

	// creting the gateway server for http requests
	gwServer := &http.Server{
		Addr:    ":" + config.HttpPort,
		Handler: mux,
	}

	log.Println("Serving gRPC-Gateway on " + httpAddress)
	log.Println("Swagger running on " + httpAddress + "/swagger-ui/")
	// creating an http listener
	log.Fatalln(gwServer.ListenAndServe())
}
