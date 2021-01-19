properties(
  [
    office365ConnectorWebhooks([[
        startNotification: true,
        notifySuccess: true,
        notifyAborted: true,
        notifyNotBuilt: true,
        notifyUnstable: true,
        notifyFailure: true,
        notifyBackToNormal: true,
        notifyRepeatedFailure: true,
        url: 'https://outlook.office.com/webhook/0899a847-7845-4e6b-ab58-8523e4ad4052@6afda026-de71-4468-92ef-11f4bcc200bf/JenkinsCI/eaf5860dc8754b7890b0c116f7e9b6df/fb3124f9-a19d-4220-afdb-d3b2bccca5df'
        ]]),
    pipelineTriggers([githubPush()])
  ]
)

pipeline {
    agent {
        label 'jenkins-slave'
    }
    environment {
        //Def
        IMAGE_TAG = 'none'
    }
    stages {
        stage('Setup'){
            steps{
                script{
                    switch (env.BRANCH_NAME) {
                        case 'master':
                            IMAGE_TAG = "${env.BRANCH_NAME}-b${env.BUILD_ID}-${GIT_COMMIT[0..6]}"
                            break
                        default:
                            IMAGE_TAG = "${env.BRANCH_NAME}"
                            break
                    }
                }
                sh """
                    apt-get update
                    apt-get install build-essential -y
                """
            }
        }
        stage('Build') {
            steps {
                echo "Build guac"
                sh """
                    make TAG=$IMAGE_TAG jenkins-docker
                """
            }
        }
        stage('Test') {
            steps {
                echo 'Start Testing...'
            }
        }
    }
    post {
        success{
            // echo "${IMAGE_TAG} is uploaded to $ECR_REGISTRY/$ECR_REPOSITORY:$IMAGE_TAG"
            jiraSendBuildInfo site: 'appaegis.atlassian.net'
        }
        cleanup {
            cleanWs()
        }
    }
}