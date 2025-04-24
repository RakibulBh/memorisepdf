# Setting Up Vercel Blob Storage for Large File Uploads

This application uses Vercel Blob Storage to handle file uploads larger than Vercel Serverless Functions' 4.5MB limit. Follow these instructions to set up your environment.

## Why Vercel Blob?

Vercel Serverless Functions have a 4.5MB limit for both request and response payloads. When uploading large files (such as PDFs or PowerPoint presentations), we need to:

1. Upload files directly from the client to Vercel Blob storage
2. Use the stored file URL to process the content on the server
3. This approach bypasses the 4.5MB serverless function limit

## Setup Instructions

### 1. Create a Vercel Blob Store

1. Log in to your Vercel account and go to the [Dashboard](https://vercel.com/dashboard)
2. Navigate to Storage → Create Blob Store
3. Name your blob store and select the desired region
4. Click "Create"

### 2. Get Your Blob Credentials

1. In your new Blob Store, click on "Connect" on the top right
2. You'll see two important values:
   - `BLOB_READ_WRITE_TOKEN`: A token for reading and writing to the blob store
   - `BLOB_STORE_ID`: Your unique blob store identifier

### 3. Add Environment Variables

Add these values to your `.env` file locally:

```
BLOB_READ_WRITE_TOKEN=your_token_here
BLOB_STORE_ID=your_store_id_here
```

For production:

1. Go to your Vercel project settings
2. Navigate to Environment Variables
3. Add the same variables with your production values

### 4. File Size Limits

The application has a client-side file size limit of 75MB. While Vercel Blob supports files up to 500MB, we've set a lower limit to optimize performance and storage usage. Files are automatically deleted from Blob storage after processing.

### 5. Testing Locally with Vercel Blob

To test Vercel Blob locally, you need to:

1. Make sure your local environment has the proper Vercel Blob credentials
2. Be aware that the `onUploadCompleted` webhook won't work locally with localhost
3. For full local testing, use a tunneling service like ngrok

## Troubleshooting

- **413 Payload Too Large**: This means you're still trying to upload through the serverless function instead of directly to Blob storage
- **Failed Uploads**: Check that your `BLOB_READ_WRITE_TOKEN` has the correct permissions
- **Webhook Issues**: In production, ensure your function returns a 200 status for the webhook to complete successfully

## Additional Resources

- [Vercel Blob Documentation](https://vercel.com/docs/storage/vercel-blob)
- [Client Upload Example](https://github.com/vercel/storage/tree/main/examples/nextjs-blob)
