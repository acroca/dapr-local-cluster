using Dapr;
using Dapr.Messaging.PublishSubscribe;
using Microsoft.AspNetCore.Mvc;
using System.Text.Json.Serialization;
using Dapr.Client;

namespace Worker.Controllers
{
    [ApiController]
    [Route("[controller]")]
    public class NewFileUploadedController : ControllerBase
    {
        private readonly ILogger<NewFileUploadedController> _logger;
        private static Random rng = new Random();

        public NewFileUploadedController(ILogger<NewFileUploadedController> logger)
        {
            _logger = logger;
        }

        [Topic("pubsub", "NewFileUploaded")]
        [HttpPost("NewFileUploaded")]
        public async Task<IActionResult> Post([FromBody] FileUpload fileUpload, DaprClient daprClient)
        {
            _logger.LogInformation($"Got page from broker: {fileUpload.FileName}, {fileUpload.PageNumber}");
            int delayDuration = rng.Next(250);
            await Task.Delay(delayDuration);
            FileUploadProcessed uploadProcessedMessage = new(
                RequestId: fileUpload.RequestId,
                DocumentId: fileUpload.DocumentId,
                PageId: fileUpload.PageId
            );

            var client = daprClient.CreateInvokableHttpClient(appId: "dsnext-api");
            var response = await client.PostAsJsonAsync("/FileUploadProcessed", uploadProcessedMessage);
            _logger.LogInformation($"page {fileUpload.FileName}, {fileUpload.PageNumber} done. Sent to dsnext-api: {response.StatusCode}");
            return Ok();
        }

        public record FileUpload([property: JsonPropertyName("fileName")] string FileName,
            [property: JsonPropertyName("pageNumber")] int PageNumber,
            [property: JsonPropertyName("binaryData")] string BinaryData,
            [property: JsonPropertyName("requestId")] string RequestId,
            [property: JsonPropertyName("documentId")] string DocumentId,
            [property: JsonPropertyName("pageId")] string PageId
        );

        public record FileUploadProcessed(
            [property: JsonPropertyName("requestId")]
            string RequestId,
            [property: JsonPropertyName("documentId")]
            string DocumentId,
            [property: JsonPropertyName("pageId")]
            string PageId
        );
    }

}
