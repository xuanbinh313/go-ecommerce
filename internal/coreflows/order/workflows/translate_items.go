package workflows

import (
	"context"
	"strings"

	productmodels "github.com/example/medusa-go-clone/internal/modules/product/domain/models"
	translationmodels "github.com/example/medusa-go-clone/internal/modules/translation/domain/models"
	ordermodels "github.com/example/medusa-go-clone/internal/modules/order/domain/models"
)

const UpdateOrderItemsTranslationsStepID = "update-order-items-translations"

type ProductReader interface { GetProduct(context.Context,string)(productmodels.Product,error) }
type TranslationReader interface { List(context.Context,translationmodels.Filter)([]translationmodels.Translation,int,error) }

func PrepareOrderItemTranslations(ctx context.Context, items []ordermodels.Item, locale string, products ProductReader, translations TranslationReader) ([]ordermodels.LineItemTranslationUpdate,error) {
	productByID:=map[string]productmodels.Product{};referenceIDs:=map[string]struct{}{}
	for _,item:=range items {
		if item.Item.ProductID==nil||*item.Item.ProductID==""{continue}
		productID:=*item.Item.ProductID
		product,ok:=productByID[productID];if !ok{var err error;product,err=products.GetProduct(ctx,productID);if err!=nil{continue};productByID[productID]=product}
		referenceIDs[productID]=struct{}{};if item.Item.VariantID!=nil{referenceIDs[*item.Item.VariantID]=struct{}{}};if product.TypeID!=nil{referenceIDs[*product.TypeID]=struct{}{}};if product.CollectionID!=nil{referenceIDs[*product.CollectionID]=struct{}{}}
	}
	ids:=make([]string,0,len(referenceIDs));for id:=range referenceIDs{ids=append(ids,id)};if len(ids)==0{return nil,nil}
	translated:=map[string]map[string]string{}
	for start:=0;start<len(ids);start+=250{end:=start+250;if end>len(ids){end=len(ids)};rows,_,err:=translations.List(ctx,translationmodels.Filter{ReferenceIDs:ids[start:end],LocaleCodes:[]string{locale},Limit:250,Offset:0});if err!=nil{return nil,err};for _,row:=range rows{translated[row.ReferenceID]=row.Translations}}
	textValue:=func(current *string, record map[string]string,key string)*string{if value,ok:=record[key];ok&&strings.TrimSpace(value)!=""{v:=value;return &v};return current}
	result:=make([]ordermodels.LineItemTranslationUpdate,0,len(items))
	for _,item:=range items{
		line:=item.Item;product,hasProduct:=productmodels.Product{},false;if line.ProductID!=nil{product,hasProduct=productByID[*line.ProductID]}
		productTranslation:=map[string]string{};variantTranslation:=map[string]string{};typeTranslation:=map[string]string{};collectionTranslation:=map[string]string{}
		if line.ProductID!=nil{productTranslation=translated[*line.ProductID]};if line.VariantID!=nil{variantTranslation=translated[*line.VariantID]};if hasProduct&&product.TypeID!=nil{typeTranslation=translated[*product.TypeID]};if hasProduct&&product.CollectionID!=nil{collectionTranslation=translated[*product.CollectionID]}
		productTitle:=textValue(line.ProductTitle,productTranslation,"title");productDescription:=textValue(line.ProductDescription,productTranslation,"description");productSubtitle:=textValue(line.ProductSubtitle,productTranslation,"subtitle");productType:=textValue(line.ProductType,typeTranslation,"value");productCollection:=textValue(line.ProductCollection,collectionTranslation,"title");productHandle:=textValue(line.ProductHandle,productTranslation,"handle");variantTitle:=textValue(line.VariantTitle,variantTranslation,"title")
		title:=line.Title;if productTitle!=nil{title=*productTitle}
		result=append(result,ordermodels.LineItemTranslationUpdate{ID:line.ID,Title:title,Subtitle:line.Subtitle,ProductTitle:productTitle,ProductDescription:productDescription,ProductSubtitle:productSubtitle,ProductType:productType,ProductCollection:productCollection,ProductHandle:productHandle,VariantTitle:variantTitle})
	}
	return result,nil
}
