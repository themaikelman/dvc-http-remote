# Unnotified error when pushing data into HTTP remote

This issue happens when pushing a bulk of files into a HTTP dvc remote. dvc push reports that everything is correct. 
However, when downloading the files, some of them have not been uploaded correactly and thus, they do not exist on the remote.

Provided script do the following:
- download a dataset
- adds it to dvc
- push the data (report_ _push.log --> No error)
- remove the cache and tmp folder inside .dvc to assure we will download the data from remote
- pull the data again (report_ _pull.log --> Some files missing)
- Push the data again (report_ _push2.log --> Everything updated!)

As it can be seen error when uploading files is not reported by dvc. Besides this, DVC thinks that everything is correctly uploaded.

```bash
export DATASET_FOLDER=cars_train 
export REMOTE_NAME=localhost

# Download random dataset
wget http://ai.stanford.edu/~jkrause/car196/cars_train.tgz
tar -xf cars_train.tgz 
rm cars_train.tgz 
export REMOTE_NAME_FILE="${REMOTE_NAME/"-"/"_"}" 

# Try with HTTP local remote:
dvc init
dvc remote add -d localhost http://localhost:8080/remote?remote=0
dvc remote modify localhost ssl_verify false
## Add and push the data
dvc add $DATASET_FOLDER
dvc push -v $DATASET_FOLDER > report_${REMOTE_NAME_FILE}_push.log 2>&1

## Download the data and check
dvc remote default ${REMOTE_NAME}
rm -rf $DATASET_FOLDER
rm -rf .dvc/cache
rm -rf .dvc/tmp
dvc pull -v $DATASET_FOLDER > report_${REMOTE_NAME_FILE}_pull.log 2>&1

##Try to push again
dvc push -v $DATASET_FOLDER > report_${REMOTE_NAME_FILE}_push_retry.log 2>&1
```

Additionally, when you try to push the files again, the .dir optimization precludes to upload again the files and dvc thinks that everything is uploaded. If the dataset have subfolders, the problem is even worse, as re-adding the files do not correct the issue due to .dir optimization.