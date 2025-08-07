using API.Models;
using Dapr;
using Dapr.Client;
using Google.Api;
using Microsoft.AspNetCore.Mvc;
using Microsoft.Extensions.Logging;
using System.Text.Json.Serialization;
using UglyToad.PdfPig;
using UglyToad.PdfPig.Writer;

namespace API.Controllers
{
    [ApiController]
    [Route("[controller]")]
    public class FilesController : ControllerBase
    {
        private readonly ILogger<FilesController> _logger;
        private static Dictionary<string, UploadRequestContext> requestContextMap = new();

        public FilesController(ILogger<FilesController> logger)
        {
            _logger = logger;
        }

        [HttpPost]
        public async Task<IActionResult> Post(IFormFileCollection files, DaprClient daprClient)
        {
            if (files.Count == 0)
            {
                _logger.LogWarning("No files were uploaded");
                return BadRequest("No files were uploaded");
            }

            string requestId = Guid.NewGuid().ToString();
            UploadRequestContext currentContext = new UploadRequestContext()
            {
                Id = requestId,
            };

            string[] fileRequestIds = new string[files.Count];
            for (int i = 0; i < files.Count; i++)
            {
                string fileGuid = Guid.NewGuid().ToString();
                fileRequestIds[i] = fileGuid;
                currentContext.FileContexts.TryAdd(fileGuid, new UploadFileContext()
                {
                    Id = fileGuid
                });
            }

            requestContextMap[requestId] = currentContext;

            var fileExtensions = new List<string>();
            long totalFileSize = 0;
            List<Task> publishTasks = new List<Task>();
            for (int i = 0; i < files.Count; i++)
            {
                IFormFile file = files[i];
                UploadFileContext fileContext = currentContext.FileContexts[fileRequestIds[i]];
                fileContext.FileName = file.FileName;
                if (file.Length > 0)
                {
                    totalFileSize += file.Length;
                    var extension = Path.GetExtension(file.FileName)?.ToLowerInvariant();
                    if (!string.IsNullOrEmpty(extension))
                    {
                        fileExtensions.Add(extension);
                    }
                }

                MemoryStream ms = new MemoryStream();
                file.OpenReadStream().CopyTo(ms);
                ms.Position = 0L;
                PdfDocument doc = PdfDocument.Open(ms);
                
                try
                {
                    for (int j = 1; j <= doc.NumberOfPages; j++)
                    {
                        string pageGuid = Guid.NewGuid().ToString();
                        fileContext.PageContexts.TryAdd(pageGuid, new()
                        {
                            Id = pageGuid,
                            PageNumber = j,
                        });
                        PdfDocumentBuilder builder = new PdfDocumentBuilder();
                        builder.AddPage(doc, j);
                        string base64 = Convert.ToBase64String(builder.Build());
                        FileUpload brokerMsg = new FileUpload(FileName: file.FileName, PageNumber: j,
                            BinaryData: base64, RequestId: currentContext.Id, DocumentId: fileContext.Id,
                            PageId: pageGuid);
                        publishTasks.Add(daprClient.PublishEventAsync("pubsub", "NewFileUploaded", brokerMsg));
                    }
                }
                catch (Exception e)
                {  
                    _logger.LogError($"Task.Run {e.Message}");
                    throw;
                }
                doc.Dispose();
                ms.Dispose();
            }
            await Task.WhenAll(publishTasks);

            var extensionGroups = fileExtensions
                .GroupBy(ext => ext)
                .Select(g => new { Extension = g.Key, Count = g.Count() })
                .ToList();

            _logger.LogInformation("FileUpload processed: {FileCount} files, Total size: {TotalSize} bytes, Extensions: {Extensions}",
                files.Count,
                totalFileSize,
                string.Join(", ", extensionGroups.Select(eg => $"{eg.Extension} ({eg.Count})")));

            return Ok(new
            {
                FileCount = files.Count,
                TotalSizeBytes = totalFileSize,
                TotalSizeMB = Math.Round(totalFileSize / (1024.0 * 1024.0), 2),
                Extensions = extensionGroups.Select(eg => new { Extension = eg.Extension, Count = eg.Count })
            });
        }

        [HttpPost("/FileUploadProcessed")]
        public IActionResult FileUploadProcessed([FromBody] FileUploadProcessed fileUploadProcessed)
        {
            _logger.LogInformation("Got FileUploadProcessed.");
            if (!requestContextMap.TryGetValue(fileUploadProcessed.RequestId, out UploadRequestContext? currentContext))
            {
                _logger.LogWarning($"Got progress update for unknown request: {fileUploadProcessed.RequestId}");
                return Ok();
            }

            if (!currentContext.FileContexts.TryGetValue(fileUploadProcessed.DocumentId,
                    out UploadFileContext? fileContext))
            {
                _logger.LogWarning($"Got progress update for known request {fileUploadProcessed.RequestId} but unknown document: {fileUploadProcessed.DocumentId}");
                return Ok();
            }

            if (!fileContext.PageContexts.TryGetValue(fileUploadProcessed.PageId,
                    out UploadPageContext? pageContext))
            {
                _logger.LogWarning($"Got progress update for known request {fileUploadProcessed.RequestId} and document: {fileUploadProcessed.DocumentId}, but unknown page {fileUploadProcessed.PageId}");
                return Ok();
            }

            pageContext.Finished = true;
            _logger.LogInformation($"page {pageContext.PageNumber} finished for {fileContext.FileName}. File Done {fileContext.Finished}, Upload Done {currentContext.Finished}");
            return Ok();
        }
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
        [property: JsonPropertyName("pageId")] string PageId
    );
}
