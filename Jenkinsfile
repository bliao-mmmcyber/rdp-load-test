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
        url: 'https://appaccacom.webhook.office.com/webhookb2/0899a847-7845-4e6b-ab58-8523e4ad4052@6afda026-de71-4468-92ef-11f4bcc200bf/JenkinsCI/c801105fc3074b8096ca4cbb39822dd3/d5d3f3be-8856-4e63-8678-e2ab8281a59d'
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
                    make VERSION=$IMAGE_TAG docker
                    make VERSION=$IMAGE_TAG docker-transcode
                """
                script {
                    if (env.BRANCH_NAME == 'master') {
                        sh(script: "docker tag 980993447824.dkr.ecr.us-east-1.amazonaws.com/appaegis/guac:${IMAGE_TAG} 980993447824.dkr.ecr.us-east-1.amazonaws.com/appaegis/guac:latest")
                        sh(script: "docker push 980993447824.dkr.ecr.us-east-1.amazonaws.com/appaegis/guac:latest")
                        sh(script: "docker tag 980993447824.dkr.ecr.us-east-1.amazonaws.com/appaegis/rdp-transcode:${IMAGE_TAG} 980993447824.dkr.ecr.us-east-1.amazonaws.com/appaegis/rdp-transcode:latest")
                        sh(script: "docker push 980993447824.dkr.ecr.us-east-1.amazonaws.com/appaegis/rdp-transcode:latest")
                    }
                }
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