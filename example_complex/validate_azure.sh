export DATASET_FOLDER=cars_train 
export REMOTE_NAME=azure

# Download random dataset
wget http://ai.stanford.edu/~jkrause/car196/cars_train.tgz
tar -xf cars_train.tgz 
rm cars_train.tgz 
export REMOTE_NAME_FILE="${REMOTE_NAME/"-"/"_"}" 

# Try with HTTP azure remote:
git init
dvc init
dvc remote add -d ${REMOTE_NAME} http://localhost:8080/remote?remote=1
dvc remote modify ${REMOTE_NAME} ssl_verify false
## Add and push the data
dvc add $DATASET_FOLDER
dvc push -v $DATASET_FOLDER > report_azure_${REMOTE_NAME_FILE}_push.log 2>&1

## Download the data and check
rm -rf $DATASET_FOLDER
rm -rf .dvc/cache
rm -rf .dvc/tmp
dvc pull -v $DATASET_FOLDER > report_azure_${REMOTE_NAME_FILE}_pull.log 2>&1

##Try to push again
dvc push -v $DATASET_FOLDER > report_azure_${REMOTE_NAME_FILE}_push_retry.log 2>&1

