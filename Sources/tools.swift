import AppKit
import Foundation
import JSONSchemaBuilder
@preconcurrency import MCPServer

/// Error type for tool operations
struct ToolError: Error {
    let message: String
}

// MARK: - Unsplash Photo Search Tool

/// Represents an Unsplash photo with its metadata
@Schemable
struct UnsplashPhoto: Codable {
    let id: String
    let description: String?
    let urls: [String: String]
    let width: Int
    let height: Int
}

/// Input parameters for searching Unsplash photos
@Schemable
struct SearchPhotosInput {
    @SchemaOptions(
        description: "Search keyword"
    )
    let query: String

    @SchemaOptions(
        description: "Page number (1-based)",
        default: 1
    )
    let page: Int?

    @SchemaOptions(
        description: "Results per page (1-30)",
        default: 10
    )
    let perPage: Int?

    @SchemaOptions(
        description: "Sort method (relevant or latest)",
        default: "relevant"
    )
    let orderBy: String?

    @SchemaOptions(
        description:
            "Color filter (black_and_white, black, white, yellow, orange, red, purple, magenta, green, teal, blue)"
    )
    let color: String?

    @SchemaOptions(
        description: "Orientation filter (landscape, portrait, squarish)"
    )
    let orientation: String?
}

let searchPhotosTool = Tool(
    name: "search_photos",
    description: "Search for Unsplash photos"
) { (input: SearchPhotosInput) async throws -> [TextContentOrImageContentOrEmbeddedResource] in
    mcpLogger.info("Starting photo search with query: \(input.query)")

    // Get access key from environment
    guard let accessKey = ProcessInfo.processInfo.environment["UNSPLASH_ACCESS_KEY"] else {
        mcpLogger.error("Missing UNSPLASH_ACCESS_KEY environment variable")
        throw ToolError(message: "Missing UNSPLASH_ACCESS_KEY environment variable")
    }

    // Build URL components
    var urlComponents = URLComponents(string: "https://api.unsplash.com/search/photos")!
    var queryItems: [URLQueryItem] = [
        URLQueryItem(name: "query", value: input.query),
        URLQueryItem(name: "page", value: String(input.page ?? 1)),
        URLQueryItem(name: "per_page", value: String(min(input.perPage ?? 10, 30))),
        URLQueryItem(name: "order_by", value: input.orderBy ?? "relevant"),
    ]

    // Add optional parameters
    if let color = input.color {
        queryItems.append(URLQueryItem(name: "color", value: color))
    }
    if let orientation = input.orientation {
        queryItems.append(URLQueryItem(name: "orientation", value: orientation))
    }

    urlComponents.queryItems = queryItems

    guard let url = urlComponents.url else {
        mcpLogger.error("Failed to construct URL")
        throw ToolError(message: "Failed to construct URL")
    }

    mcpLogger.debug("Making API request to: \(url.absoluteString)")

    // Create URL request
    var request = URLRequest(url: url)
    request.setValue("v1", forHTTPHeaderField: "Accept-Version")
    request.setValue("Client-ID \(accessKey)", forHTTPHeaderField: "Authorization")

    // Perform request
    let (data, response) = try await URLSession.shared.data(for: request)

    guard let httpResponse = response as? HTTPURLResponse else {
        mcpLogger.error("Invalid response type received")
        throw ToolError(message: "Invalid response type")
    }

    guard httpResponse.statusCode == 200 else {
        mcpLogger.error("HTTP error: \(httpResponse.statusCode)")
        throw ToolError(message: "HTTP error: \(httpResponse.statusCode)")
    }

    // Decode response
    struct SearchResponse: Codable {
        let results: [UnsplashPhoto]
    }

    do {
        let decoder = JSONDecoder()
        let searchResponse = try decoder.decode(SearchResponse.self, from: data)
        mcpLogger.info("Successfully found \(searchResponse.results.count) photos")

        // Format response
        var responseText = "Found \(searchResponse.results.count) photos:\n\n"
        for photo in searchResponse.results {
            responseText += "- ID: \(photo.id)\n"
            if let description = photo.description {
                responseText += "  Description: \(description)\n"
            }
            responseText += "  Size: \(photo.width)x\(photo.height)\n"
            responseText += "  URLs:\n"
            for (size, url) in photo.urls {
                responseText += "    \(size): \(url)\n"
            }
            responseText += "\n"
        }

        return [.text(TextContent(text: responseText))]
    } catch {
        mcpLogger.error("Failed to decode response: \(error)")
        throw error
    }
}

// MARK: - Get Photo Tool

/// Input parameters for getting a specific photo
@Schemable
struct GetPhotoInput {
    @SchemaOptions(
        description: "The photo ID to retrieve"
    )
    let photoId: String
}

let getPhotoTool = Tool(
    name: "get_photo",
    description: "Get detailed information about a specific Unsplash photo"
) { (input: GetPhotoInput) async throws -> [TextContentOrImageContentOrEmbeddedResource] in
    mcpLogger.info("Getting photo details for ID: \(input.photoId)")

    // Get access key from environment
    guard let accessKey = ProcessInfo.processInfo.environment["UNSPLASH_ACCESS_KEY"] else {
        mcpLogger.error("Missing UNSPLASH_ACCESS_KEY environment variable")
        throw ToolError(message: "Missing UNSPLASH_ACCESS_KEY environment variable")
    }

    // Build URL
    guard let url = URL(string: "https://api.unsplash.com/photos/\(input.photoId)") else {
        mcpLogger.error("Failed to construct URL for photo ID: \(input.photoId)")
        throw ToolError(message: "Failed to construct URL")
    }

    mcpLogger.debug("Making API request to: \(url.absoluteString)")

    // Create URL request
    var request = URLRequest(url: url)
    request.setValue("v1", forHTTPHeaderField: "Accept-Version")
    request.setValue("Client-ID \(accessKey)", forHTTPHeaderField: "Authorization")

    // Perform request
    let (data, response) = try await URLSession.shared.data(for: request)

    guard let httpResponse = response as? HTTPURLResponse else {
        mcpLogger.error("Invalid response type received")
        throw ToolError(message: "Invalid response type")
    }

    guard httpResponse.statusCode == 200 else {
        mcpLogger.error("HTTP error: \(httpResponse.statusCode)")
        throw ToolError(message: "HTTP error: \(httpResponse.statusCode)")
    }

    // Define detailed photo response structure
    struct DetailedPhoto: Codable {
        let id: String
        let description: String?
        let altDescription: String?
        let urls: [String: String]
        let width: Int
        let height: Int
        let likes: Int
        let downloads: Int?
        let location: Location?
        let exif: Exif?
        let user: User
        let tags: [Tag]?

        struct Location: Codable {
            let name: String?
            let city: String?
            let country: String?
        }

        struct Exif: Codable {
            let make: String?
            let model: String?
            let exposureTime: String?
            let aperture: String?
            let focalLength: String?
            let iso: Int?
        }

        struct User: Codable {
            let name: String
            let username: String
            let portfolioUrl: String?
        }

        struct Tag: Codable {
            let title: String
        }
    }

    do {
        // Decode response
        let decoder = JSONDecoder()
        decoder.keyDecodingStrategy = .convertFromSnakeCase
        let photo = try decoder.decode(DetailedPhoto.self, from: data)
        mcpLogger.info("Successfully retrieved photo details for ID: \(input.photoId)")

        // Format response
        var responseText = "Photo Details:\n\n"
        responseText += "- ID: \(photo.id)\n"
        if let description = photo.description {
            responseText += "- Description: \(description)\n"
        }
        if let altDescription = photo.altDescription {
            responseText += "- Alt Description: \(altDescription)\n"
        }
        responseText += "- Size: \(photo.width)x\(photo.height)\n"
        responseText += "- Likes: \(photo.likes)\n"
        if let downloads = photo.downloads {
            responseText += "- Downloads: \(downloads)\n"
        }

        responseText += "\nPhotographer:\n"
        responseText += "- Name: \(photo.user.name)\n"
        responseText += "- Username: \(photo.user.username)\n"
        if let portfolioUrl = photo.user.portfolioUrl {
            responseText += "- Portfolio: \(portfolioUrl)\n"
        }

        if let location = photo.location, location.name != nil || location.city != nil || location.country != nil {
            responseText += "\nLocation:\n"
            if let name = location.name {
                responseText += "- Name: \(name)\n"
            }
            if let city = location.city {
                responseText += "- City: \(city)\n"
            }
            if let country = location.country {
                responseText += "- Country: \(country)\n"
            }
        }

        if let exif = photo.exif, !Mirror(reflecting: exif).children.allSatisfy({ $0.value as? String == nil }) {
            responseText += "\nCamera Info:\n"
            if let make = exif.make {
                responseText += "- Camera Make: \(make)\n"
            }
            if let model = exif.model {
                responseText += "- Camera Model: \(model)\n"
            }
            if let exposureTime = exif.exposureTime {
                responseText += "- Exposure Time: \(exposureTime)\n"
            }
            if let aperture = exif.aperture {
                responseText += "- Aperture: \(aperture)\n"
            }
            if let focalLength = exif.focalLength {
                responseText += "- Focal Length: \(focalLength)\n"
            }
            if let iso = exif.iso {
                responseText += "- ISO: \(iso)\n"
            }
        }

        responseText += "\nURLs:\n"
        for (size, url) in photo.urls {
            responseText += "- \(size): \(url)\n"
        }

        if let tags = photo.tags, !tags.isEmpty {
            responseText += "\nTags:\n"
            for tag in tags {
                responseText += "- \(tag.title)\n"
            }
        }

        return [.text(TextContent(text: responseText))]
    } catch {
        mcpLogger.error("Failed to decode photo details: \(error)")
        throw error
    }
}

// MARK: - Random Photo Tool

/// Input parameters for getting random photos
@Schemable
struct RandomPhotoInput {
    @SchemaOptions(
        description: "The number of photos to return (Default: 1; Max: 30)",
        default: 1
    )
    let count: Int?

    @SchemaOptions(
        description: "Public collection ID('s) to filter selection. If multiple, comma-separated"
    )
    let collections: String?

    @SchemaOptions(
        description: "Public topic ID('s) to filter selection. If multiple, comma-separated"
    )
    let topics: String?

    @SchemaOptions(
        description: "Limit selection to a specific user"
    )
    let username: String?

    @SchemaOptions(
        description: "Limit selection to photos matching a search term"
    )
    let query: String?

    @SchemaOptions(
        description: "Filter by photo orientation. Valid values: landscape, portrait, squarish"
    )
    let orientation: String?

    @SchemaOptions(
        description: "Limit results by content safety. Valid values: low, high"
    )
    let contentFilter: String?

    @SchemaOptions(
        description: "Limit selection to featured photos"
    )
    let featured: Bool?
}

let randomPhotoTool = Tool(
    name: "random_photo",
    description: "Get one or more random photos from Unsplash"
) { (input: RandomPhotoInput) async throws -> [TextContentOrImageContentOrEmbeddedResource] in
    mcpLogger.info("Starting random photo request with count: \(input.count ?? 1)")

    // Get access key from environment
    guard let accessKey = ProcessInfo.processInfo.environment["UNSPLASH_ACCESS_KEY"] else {
        mcpLogger.error("Missing UNSPLASH_ACCESS_KEY environment variable")
        throw ToolError(message: "Missing UNSPLASH_ACCESS_KEY environment variable")
    }

    // Build URL components
    var urlComponents = URLComponents(string: "https://api.unsplash.com/photos/random")!
    var queryItems: [URLQueryItem] = []

    // Add parameters
    if let count = input.count {
        queryItems.append(URLQueryItem(name: "count", value: String(min(count, 30))))
    }
    if let query = input.query {
        queryItems.append(URLQueryItem(name: "query", value: query))
    }
    if let collections = input.collections {
        queryItems.append(URLQueryItem(name: "collections", value: collections))
    }
    if let orientation = input.orientation {
        queryItems.append(URLQueryItem(name: "orientation", value: orientation))
    }
    if let featured = input.featured {
        queryItems.append(URLQueryItem(name: "featured", value: String(featured)))
    }
    if let username = input.username {
        queryItems.append(URLQueryItem(name: "username", value: username))
    }
    if let topics = input.topics {
        queryItems.append(URLQueryItem(name: "topics", value: topics))
    }
    if let contentFilter = input.contentFilter {
        queryItems.append(URLQueryItem(name: "content_filter", value: contentFilter))
    }

    urlComponents.queryItems = queryItems

    guard let url = urlComponents.url else {
        mcpLogger.error("Failed to construct URL for random photos")
        throw ToolError(message: "Failed to construct URL")
    }

    mcpLogger.debug("Making API request to: \(url.absoluteString)")

    // Create URL request
    var request = URLRequest(url: url)
    request.setValue("v1", forHTTPHeaderField: "Accept-Version")
    request.setValue("Client-ID \(accessKey)", forHTTPHeaderField: "Authorization")

    // Perform request
    let (data, response) = try await URLSession.shared.data(for: request)

    guard let httpResponse = response as? HTTPURLResponse else {
        mcpLogger.error("Invalid response type received")
        throw ToolError(message: "Invalid response type")
    }

    guard httpResponse.statusCode == 200 else {
        mcpLogger.error("HTTP error: \(httpResponse.statusCode)")
        throw ToolError(message: "HTTP error: \(httpResponse.statusCode)")
    }

    do {
        // Decode response
        let decoder = JSONDecoder()
        decoder.keyDecodingStrategy = .convertFromSnakeCase

        // Response can be either a single photo or an array of photos
        let photos: [UnsplashPhoto]
        if input.count == nil || input.count == 1 {
            let photo = try decoder.decode(UnsplashPhoto.self, from: data)
            photos = [photo]
        } else {
            photos = try decoder.decode([UnsplashPhoto].self, from: data)
        }

        mcpLogger.info("Successfully retrieved \(photos.count) random photos")

        // Format response
        var responseText = "Random Photos:\n\n"
        for (index, photo) in photos.enumerated() {
            responseText += "Photo \(index + 1):\n"
            responseText += "- ID: \(photo.id)\n"
            if let description = photo.description {
                responseText += "- Description: \(description)\n"
            }
            responseText += "- Size: \(photo.width)x\(photo.height)\n"
            responseText += "- URLs:\n"
            for (size, url) in photo.urls {
                responseText += "  \(size): \(url)\n"
            }
            responseText += "\n"
        }

        return [.text(TextContent(text: responseText))]
    } catch {
        mcpLogger.error("Failed to decode random photos response: \(error)")
        throw error
    }
}
