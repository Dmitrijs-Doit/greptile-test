/* eslint-disable no-console */
const util = require("util");
const https = require("https");
const response = require("cfn-response-promise");

async function replySucessToResourceRequest(event, context) {
  return response.send(event, context, response.SUCCESS, {});
}
async function replyFailToResourceRequest(event, context) {
  return response.send(event, context, response.FAILED, {});
}

module.exports.handler = async (event, context) => {
  let dataString = "";
  console.log(`Got SNS message: ${util.inspect(event, { depth: 5 })}`);
  const snsMessage = event.Records[0].Sns;
  const resourceEvent = JSON.parse(snsMessage.Message);
  const notificationUrl = resourceEvent.ResourceProperties.NotificationUrl;

  console.log("flow is: ", resourceEvent.RequestType.toLowerCase());
  if (resourceEvent.RequestType.toLowerCase() !== "create" && resourceEvent.RequestType.toLowerCase() !== "delete") {
    return response.send(event, context, response.SUCCESS, {});
  }
  const data = JSON.stringify({
    stack_id: resourceEvent.StackId,
    management_arn: resourceEvent.ResourceProperties.RoleArn,
    external_id: resourceEvent.ResourceProperties.ExternalID,
    account_id: resourceEvent.ResourceProperties.AccountID,
    cur_path: resourceEvent.ResourceProperties.CurPath,
    s3_bucket: resourceEvent.ResourceProperties.S3Bucket,
  });

  console.log(`Sending notification to CMP. url: ${notificationUrl}, data: ${data}`);

  const url = new URL(notificationUrl);

  const options = {
    hostname: url.hostname,
    path: url.pathname,
    method: resourceEvent.RequestType && resourceEvent.RequestType.toLowerCase() === "delete" ? "POST" : "PUT",
    headers: {
      "Content-Type": "application/json",
    },
  };

  const response2 = await new Promise((resolve) => {
    const req = https.request(options, (res) => {
      res.on("data", (chunk) => {
        dataString += chunk;
        console.log("cmp results:", res.statusCode, " ", res.body);
      });
      res.on("end", () => {
        console.log("end ", dataString);
        console.log("end ", res.statusCode, " ", res.body);
        resolve({
          statusCode: res.statusCode,
          body: res.body,
        });
      });
    });

    req.on("error", (err) => {
      console.log("got an error from cmp", err);
      resolve({
        statusCode: 400,
        body: "error",
      });
    });
    req.write(data);
    req.end();
  });
  if (response2.statusCode === 200) {
    await replySucessToResourceRequest(resourceEvent, context);
  } else {
    await replyFailToResourceRequest(resourceEvent, context);
  }
};
