import GenericToolInterface from "@/components/studio/GenericToolInterface";

export default async function GenericToolPage({ params }: { params: Promise<{ id: string }> }) {
  const resolvedParams = await params;
  return <GenericToolInterface params={resolvedParams} />;
}
