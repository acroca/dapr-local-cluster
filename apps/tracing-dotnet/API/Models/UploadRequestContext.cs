using System.Collections.Concurrent;

namespace API.Models
{
    public class UploadRequestContext
    {
        public required string Id;
        public bool Finished => FileContexts.Values.All(fc => fc.Finished);
        public ConcurrentDictionary<string, UploadFileContext> FileContexts = new();
    }

    public class UploadFileContext
    {
        public required string Id;
        public bool Finished => PageContexts.Values.All(fc => fc.Finished);
        public string FileName = "";
        public ConcurrentDictionary<string, UploadPageContext> PageContexts = new();
    }

    public class UploadPageContext
    {
        public required string Id;
        public bool Finished = false;
        public int PageNumber;
    }
}
