#!/usr/bin/env groovy

def codedeploy_id
def proceed_stage = false
def proceed_prod = false

pipeline {
  agent any
  options {
    timestamps()
    ansiColor('xterm')
    buildDiscarder(logRotator(numToKeepStr: '10'))
    disableConcurrentBuilds()
  }
  parameters {
    string(defaultValue: '', description: '', name: 'DEPLOY_TAG', trim: false)
  }
  environment {
    DOCKER_CMD="docker run --rm -t `tty &>/dev/null && echo '-i'` -e \"AWS_DEFAULT_REGION=eu-west-1\" -v `pwd`:/project mesosphere/aws-cli "
    CODE_DEPLOY_CMD="lambda update-function-code --function-name \"\${ENV}-\${FUNCTION_NAME}\" --s3-bucket tfgmaws-drone-artifacts --s3-key \"TfGMEnterprise/departures-service/builds/\${FUNCTION_NAME}-\${DEPLOY_TAG}.zip\" --publish"
    slackMsg = ""
}
  stages {
   stage('Automatic Deployment to DEV') {
                  environment {
                ENV = 'dev'
            }
      steps {
        script {
          proceed_stage = true
          if(proceed_stage) {
            try {
            codedeploy_result = sh returnStdout: true, script:"""  
            FUNCTION_NAME='circular-services'
            ${DOCKER_CMD} ${CODE_DEPLOY_CMD}
             FUNCTION_NAME='ingester'
            ${DOCKER_CMD} ${CODE_DEPLOY_CMD}
            FUNCTION_NAME='locality-names'
            ${DOCKER_CMD} ${CODE_DEPLOY_CMD}
            FUNCTION_NAME='optis-poller'
            ${DOCKER_CMD} ${CODE_DEPLOY_CMD}
            FUNCTION_NAME='presenter'
            ${DOCKER_CMD} ${CODE_DEPLOY_CMD}
            FUNCTION_NAME='stops-in-area'
            ${DOCKER_CMD} ${CODE_DEPLOY_CMD}
            FUNCTION_NAME='rail-ingester'
            ${DOCKER_CMD} ${CODE_DEPLOY_CMD}            
            FUNCTION_NAME='rail-departures-board-poller'
            ${DOCKER_CMD} ${CODE_DEPLOY_CMD}            
            """
          }
          catch (err) {
              slackMsg = "Deployment of ${DEPLOY_TAG} to ${ENV} failed"
              echo "${slackMsg}"
              currentBuild.result = "FAILED"
              proceed_stage = false
          }
        }
      }
    }
  }
    stage('Approve Deployment to NFT') {
                when {
        expression { proceed_stage }
      }
                  environment {
                ENV = 'nft'
            }
      steps {
        script {
          proceed_stage = true
          try {
            timeout(time: 30, unit: 'SECONDS') {
              input(id: "Approve NFT Deployment (Y/N)", message: "Are you sure you want to deploy to NFT?", ok: 'Deploy')
            }
          } catch (err) {
              proceed_stage = false
              currentBuild.result = "NOT_BUILT"
          }
          if(proceed_stage) {
            try {
            codedeploy_result = sh returnStdout: true, script:"""  
            FUNCTION_NAME='circular-services'
            ${DOCKER_CMD} ${CODE_DEPLOY_CMD}
             FUNCTION_NAME='ingester'
            ${DOCKER_CMD} ${CODE_DEPLOY_CMD}
            FUNCTION_NAME='locality-names'
            ${DOCKER_CMD} ${CODE_DEPLOY_CMD}
            FUNCTION_NAME='optis-poller'
            ${DOCKER_CMD} ${CODE_DEPLOY_CMD}
            FUNCTION_NAME='presenter'
            ${DOCKER_CMD} ${CODE_DEPLOY_CMD}
            FUNCTION_NAME='stops-in-area'
            ${DOCKER_CMD} ${CODE_DEPLOY_CMD}
            FUNCTION_NAME='rail-ingester'
            ${DOCKER_CMD} ${CODE_DEPLOY_CMD}            
            FUNCTION_NAME='rail-departures-board-poller'
            ${DOCKER_CMD} ${CODE_DEPLOY_CMD}                            
            """
          }
          catch (err) {
              slackMsg = "Deployment of ${DEPLOY_TAG} to ${ENV} failed"
              echo "${slackMsg}"
              currentBuild.result = "FAILED"
              proceed_stage = false
          }
        }
      }
    }
  }
    stage('Approve Deployment to PRD') {
            when {
        expression { proceed_stage }
      }
                  environment {
                ENV = 'prod'
            }
      steps {
        script {
          proceed_stage = true
          try {
            timeout(time: 30, unit: 'SECONDS') {
              input(id: "Approve Production Deployment (Y/N)", message: "Are you sure you want to deploy to Production?", ok: 'Deploy')
            }
          } catch (err) {
              proceed_stage = false
              currentBuild.result = "SUCCESS"
              slackMsg = "Deployment of ${DEPLOY_TAG} to NFT passed, but deployment to PRD was cancelled"
          }
          if(proceed_stage) {
            try{
            codedeploy_result = sh returnStdout: true, script:"""  
            FUNCTION_NAME='circular-services'
            ${DOCKER_CMD} ${CODE_DEPLOY_CMD}
            FUNCTION_NAME='ingester'
            ${DOCKER_CMD} ${CODE_DEPLOY_CMD}
            FUNCTION_NAME='locality-names'
            ${DOCKER_CMD} ${CODE_DEPLOY_CMD}
            FUNCTION_NAME='optis-poller'
            ${DOCKER_CMD} ${CODE_DEPLOY_CMD}
            FUNCTION_NAME='presenter'
            ${DOCKER_CMD} ${CODE_DEPLOY_CMD}
            FUNCTION_NAME='stops-in-area'
            ${DOCKER_CMD} ${CODE_DEPLOY_CMD}   
            FUNCTION_NAME='rail-ingester'
            ${DOCKER_CMD} ${CODE_DEPLOY_CMD}            
            FUNCTION_NAME='rail-departures-board-poller'
            ${DOCKER_CMD} ${CODE_DEPLOY_CMD}                         
            """
            }
          catch (err) {
              slackMsg = "Deployment of ${DEPLOY_TAG} to NFT passed, but deployment to ${ENV} failed"
              echo "${slackMsg}"
              currentBuild.result = "FAILED"
              proceed_stage = false
          }
          slackMsg = "Deployment of ${DEPLOY_TAG} to NFT and PRD passed"
          }
        }
      }
    }
  }
     post {
         success {
             echo "Successful ${slackMsg}"
         }
         failure {
             echo "Failure ${slackMsg}"
        }
     }
}
